package cmd

import (
	"context"
	"fmt"

	"github.com/ipfs/go-log"
	"github.com/keep-network/keep-common/pkg/chain/ethereum/ethutil"
	"github.com/keep-network/keep-common/pkg/persistence"
	"github.com/keep-network/keep-tecdsa/internal/config"
	"github.com/keep-network/keep-tecdsa/pkg/chain/eth/ethereum"
	"github.com/keep-network/keep-tecdsa/pkg/client"
	"github.com/urfave/cli"
)

var logger = log.Logger("keep-cmd")

// StartCommand contains the definition of the start command-line subcommand.
var StartCommand cli.Command

const startDescription = `Starts the Keep tECDSA client in the foreground.`

func init() {
	StartCommand =
		cli.Command{
			Name:        "start",
			Usage:       `Starts the Keep tECDSA client in the foreground`,
			Description: startDescription,
			Action:      Start,
		}
}

// Start starts a client.
func Start(c *cli.Context) error {
	config, err := config.ReadConfig(c.GlobalString("config"))
	if err != nil {
		return fmt.Errorf("failed while reading config file: [%v]", err)
	}

	ethereumKey, err := ethutil.DecryptKeyFile(
		config.Ethereum.Account.KeyFile,
		config.Ethereum.Account.KeyFilePassword,
	)
	if err != nil {
		return fmt.Errorf(
			"failed to read key file [%s]: [%v]", config.Ethereum.Account.KeyFile, err,
		)
	}

	ethereumChain, err := ethereum.Connect(ethereumKey.PrivateKey, &config.Ethereum)
	if err != nil {
		return fmt.Errorf("failed to connect to ethereum node: [%v]", err)
	}

	persistence := persistence.NewEncryptedPersistence(
		persistence.NewDiskHandle(config.Storage.DataDir),
		config.Ethereum.Account.KeyFilePassword,
	)

	ctx := context.Background()

	sanctionedApplications, err := config.SanctionedApplications.Addresses()
	if err != nil {
		return fmt.Errorf("failed to get sanctioned applications addresses: [%v]", err)
	}

	client.Initialize(
		ethereumChain,
		persistence,
		sanctionedApplications,
	)

	logger.Info("client started")

	select {
	case <-ctx.Done():
		if err != nil {
			return err
		}

		return fmt.Errorf("unexpected context cancellation")
	}
}
