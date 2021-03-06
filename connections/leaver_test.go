package connections

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

func ExampleLeaver() {
	var (
		connClient *Client
		config     = &config.Config{
			Host:       "http://localhost:5001",
			APIVersion: "v1",
		}
		tokenAddress = common.HexToAddress("0x89d24a6b4ccb1b6faa2625fe562bdd9a23260359") // DAI Stablecoin
		addresses    []common.Address
		err          error
	)

	connClient = NewClient(config, http.DefaultClient)

	if addresses, err = connClient.Leave(context.Background(), tokenAddress); err != nil {
		panic(fmt.Sprintf("unable to leave connection: %s", err.Error()))
	}

	for _, a := range addresses {
		fmt.Println("address:", a.String())
	}
}

func TestLeaver(t *testing.T) {
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
		name              string
		prepHTTPMock      func()
		expectedAddresses []common.Address
		expectedError     error
	}

	testcases := []testcase{
		testcase{
			name: "successfully joined a token network",
			prepHTTPMock: func() {
				httpmock.RegisterResponder(
					"DELETE",
					"http://localhost:5001/api/v1/connections/0x2a65Aca4D5fC5B5C859090a6c34d164135398226",
					httpmock.NewStringResponder(
						http.StatusNoContent,
						`["0x41BCBC2fD72a731bcc136Cf6F7442e9C19e9f313","0x5A5f458F6c1a034930E45dC9a64B99d7def06D7E","0x8942c06FaA74cEBFf7d55B79F9989AdfC85C6b85"]`,
					),
				)
			},
			expectedAddresses: []common.Address{
				common.HexToAddress("0x41BCBC2fD72a731bcc136Cf6F7442e9C19e9f313"),
				common.HexToAddress("0x5A5f458F6c1a034930E45dC9a64B99d7def06D7E"),
				common.HexToAddress("0x8942c06FaA74cEBFf7d55B79F9989AdfC85C6b85"),
			},
			expectedError: nil,
		},
		testcase{
			name: "unexpected 500 response",
			prepHTTPMock: func() {
				httpmock.RegisterResponder(
					"DELETE",
					"http://localhost:5001/api/v1/connections/0x2a65Aca4D5fC5B5C859090a6c34d164135398226",
					httpmock.NewStringResponder(
						http.StatusInternalServerError,
						``,
					),
				)
			},
			expectedAddresses: []common.Address{},
			expectedError:     errors.New("EOF"),
		},
		testcase{
			name: "unable to make http request",
			prepHTTPMock: func() {
				httpmock.Deactivate()
			},
			expectedAddresses: []common.Address{},
			expectedError:     fmt.Errorf("Delete http://localhost:5001/api/v1/connections/0x2a65Aca4D5fC5B5C859090a6c34d164135398226: dial tcp %s:5001: connect: connection refused", localhostIP),
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			var (
				err       error
				addresses []common.Address

				tokenAddress = common.HexToAddress("0x2a65Aca4D5fC5B5C859090a6c34d164135398226")
				leaver       = NewLeaver(config, http.DefaultClient)
				ctx          = context.Background()
			)

			httpmock.Activate()
			defer httpmock.Deactivate()

			tc.prepHTTPMock()

			addresses, err = leaver.Leave(ctx, tokenAddress)

			if tc.expectedError != nil {
				assert.EqualError(t, err, tc.expectedError.Error())
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tc.expectedAddresses, addresses)
		})
	}
}
