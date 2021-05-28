package secrets

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/containers/common/pkg/completion"
	"github.com/containers/podman/v3/cmd/podman/common"
	"github.com/containers/podman/v3/cmd/podman/registry"
	"github.com/containers/podman/v3/pkg/domain/entities"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var (
	createCmd = &cobra.Command{
		Use:   "create [options] NAME FILE|-",
		Short: "Create a new secret",
		Long:  "Create a secret. Input can be a path to a file or \"-\" (read from stdin). Default driver is file (unencrypted).",
		RunE:  create,
		Args:  cobra.ExactArgs(2),
		Example: `podman secret create mysecret /path/to/secret
		printf "secretdata" | podman secret create mysecret -`,
		ValidArgsFunction: common.AutocompleteSecretCreate,
	}
)

var (
	createOpts = entities.SecretCreateOptions{}
	env        = false
)

func init() {
	registry.Commands = append(registry.Commands, registry.CliCommand{
		Command: createCmd,
		Parent:  secretCmd,
	})

	flags := createCmd.Flags()

	driverFlagName := "driver"
	flags.StringVar(&createOpts.Driver, driverFlagName, "file", "Specify secret driver")
	_ = createCmd.RegisterFlagCompletionFunc(driverFlagName, completion.AutocompleteNone)

	envFlagName := "env"
	flags.BoolVar(&env, envFlagName, false, "Read secret data from environment variable")
}

func create(cmd *cobra.Command, args []string) error {
	name := args[0]

	var err error
	path := args[1]

	var reader io.Reader
	if env {
		envValue := os.Getenv(path)
		if envValue == "" {
			return errors.Errorf("cannot create store secret data: environment variable %s is not set", path)
		}
		reader = strings.NewReader(envValue)
	} else if path == "-" || path == "/dev/stdin" {
		stat, err := os.Stdin.Stat()
		if err != nil {
			return err
		}
		if (stat.Mode() & os.ModeNamedPipe) == 0 {
			return errors.New("if `-` is used, data must be passed into stdin")
		}
		reader = os.Stdin
	} else {
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()
		reader = file
	}

	report, err := registry.ContainerEngine().SecretCreate(context.Background(), name, reader, createOpts)
	if err != nil {
		return err
	}
	fmt.Println(report.ID)
	return nil
}
