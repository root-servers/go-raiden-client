package channels

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"testing"

	"github.com/cpurta/go-raiden-client/config"
	"github.com/ethereum/go-ethereum/common"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func ExampleCloser() {
	var (
		channelClient *Client
		config        = &config.Config{
			Host:       "http://localhost:5001",
			APIVersion: "v1",
		}
		tokenAddress   = common.HexToAddress("0x89d24a6b4ccb1b6faa2625fe562bdd9a23260359") // DAI Stablecoin
		partnerAddress = common.HexToAddress("0x1f7402f55e142820ea3812106d0657103fc1709e")
		channel        *Channel
		err            error
	)

	channelClient = NewClient(config, http.DefaultClient)

	if channel, err = channelClient.Close(context.Background(), tokenAddress, partnerAddress); err != nil {
		panic(fmt.Sprintf("unable to close channel: %s", err.Error()))
	}

	fmt.Printf("Close Channel Info: %+v\n", channel)
}

func TestCloser(t *testing.T) {
	var (
		localhostIP = "[::1]"
		config      = &config.Config{
			Host:       "http://localhost:5001",
			APIVersion: "v1",
		}
	)

	if os.Getenv("USE_IPV4") != "" {
		localhostIP = "127.0.0.1"
	}

	type testcase struct {
		name            string
		prepHTTPMock    func()
		expectedChannel *Channel
		expectedError   error
	}

	testcases := []testcase{
		testcase{
			name: "successfully closed payment channel",
			prepHTTPMock: func() {
				httpmock.RegisterResponder(
					"PATCH",
					"http://localhost:5001/api/v1/channels/0xEA674fdDe714fd979de3EdF0F56AA9716B898ec8/0x61C808D82A3Ac53231750daDc13c777b59310bD9",
					httpmock.NewStringResponder(
						http.StatusOK,
						`{"token_network_identifier":"0xE5637F0103794C7e05469A9964E4563089a5E6f2","channel_identifier":20,"partner_address":"0x61C808D82A3Ac53231750daDc13c777b59310bD9","token_address":"0xEA674fdDe714fd979de3EdF0F56AA9716B898ec8","balance":25000000,"total_deposit":35000000,"state":"closed","settle_timeout":500,"reveal_timeout":30}`,
					),
				)
			},
			expectedError: nil,
			expectedChannel: &Channel{
				TokenNetworkIdentifier: common.HexToAddress("0xE5637F0103794C7e05469A9964E4563089a5E6f2"),
				ChannelIdentifier:      int64(20),
				PartnerAddress:         common.HexToAddress("0x61C808D82A3Ac53231750daDc13c777b59310bD9"),
				TokenAddress:           common.HexToAddress("0xEA674fdDe714fd979de3EdF0F56AA9716B898ec8"),
				Balance:                int64(25000000),
				TotalDeposit:           int64(35000000),
				State:                  "closed",
				SettleTimeout:          int64(500),
				RevealTimeout:          int64(30),
			},
		},
		testcase{
			name: "unexpected 500 response",
			prepHTTPMock: func() {
				httpmock.RegisterResponder(
					"PATCH",
					"http://localhost:5001/api/v1/channels/0xEA674fdDe714fd979de3EdF0F56AA9716B898ec8/0x61C808D82A3Ac53231750daDc13c777b59310bD9",
					httpmock.NewStringResponder(
						http.StatusInternalServerError,
						``,
					),
				)
			},
			expectedError:   errors.New("EOF"),
			expectedChannel: &Channel{},
		},
		testcase{
			name: "unable to make http request",
			prepHTTPMock: func() {
				httpmock.Deactivate()
			},
			expectedError:   fmt.Errorf("Patch http://localhost:5001/api/v1/channels/0xEA674fdDe714fd979de3EdF0F56AA9716B898ec8/0x61C808D82A3Ac53231750daDc13c777b59310bD9: dial tcp %s:5001: connect: connection refused", localhostIP),
			expectedChannel: &Channel{},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			var (
				err            error
				channel        *Channel
				tokenAddress   = common.HexToAddress("0xEA674fdDe714fd979de3EdF0F56AA9716B898ec8")
				partnerAddress = common.HexToAddress("0x61C808D82A3Ac53231750daDc13c777b59310bD9")

				closer = NewCloser(config, http.DefaultClient)
				ctx    = context.Background()
			)

			httpmock.Activate()
			defer httpmock.Deactivate()

			tc.prepHTTPMock()

			channel, err = closer.Close(ctx, tokenAddress, partnerAddress)

			if tc.expectedError != nil {
				assert.EqualError(t, err, tc.expectedError.Error())
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tc.expectedChannel, channel)
		})
	}
}
