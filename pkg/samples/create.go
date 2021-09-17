package samples

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/afero"

	"github.com/stripe/stripe-cli/pkg/config"
	gitpkg "github.com/stripe/stripe-cli/pkg/git"

	"gopkg.in/src-d/go-git.v4"
)

// CreationStatus is the current step in the sample creation routine
type CreationStatus int

const (
	// WillInitialize means this sample will be initialized
	WillInitialize CreationStatus = iota

	// DidInitialize means this sample has finished initializing
	DidInitialize

	// WillCopy means the downloaded sample will be copied to the target path
	WillCopy

	// DidCopy means the downloaded sample has finished being copied to the target path
	DidCopy

	// WillConfigure means the .env of the sample will be configured with the user's Stripe account details
	WillConfigure

	// DidConfigure means the .env of the sample has finished being configured with the user's Stripe account details
	DidConfigure

	// Done means sample creation is complete
	Done
)

// CreationResult is the return value sent over a channel
type CreationResult struct {
	State       CreationStatus
	Path        string
	PostInstall string
	Err         error
}

// Create creates a sample at a destination with the selected integration, client language, and server language
func Create(
	ctx context.Context,
	config *config.Config,
	sampleName string,
	selectedConfig *SelectedConfig,
	destination string,
	forceRefresh bool,
	resultChan chan<- CreationResult,
) {
	defer close(resultChan)

	sample := Samples{
		Config: config,
		Fs:     afero.NewOsFs(),
		Git:    gitpkg.Operations{},
	}

	exists, _ := afero.DirExists(sample.Fs, destination)
	if exists {
		resultChan <- CreationResult{Err: fmt.Errorf("Path already exists for: %s", destination)}
		return
	}

	if forceRefresh {
		err := sample.DeleteCache(sampleName)
		if err != nil {
			logger := log.Logger{
				Out: os.Stdout,
			}

			logger.WithFields(log.Fields{
				"prefix": "samples.create.forceRefresh",
				"error":  err,
			}).Debug("Could not clear cache")
		}
	}

	resultChan <- CreationResult{State: WillInitialize}

	// Initialize the selected sample in the local cache directory.
	// This will either clone or update the specified sample,
	// depending on whether or not it's. Additionally, this
	// identifies if the sample has multiple integrations and what
	// languages it supports.
	err := sample.Initialize(sampleName)
	if err != nil {
		switch e := err.Error(); e {
		case git.NoErrAlreadyUpToDate.Error():
			// Repo is already up to date. This isn't a program
			// error to continue as normal
			break
		case git.ErrRepositoryAlreadyExists.Error():
			// If the repository already exists and we don't pull
			// for some reason, that's fine as we can use the existing
			// repository
			break
		default:
			resultChan <- CreationResult{Err: err}
			return
		}
	}

	resultChan <- CreationResult{State: DidInitialize}

	sample.SelectedConfig = *selectedConfig

	// Setup to intercept ctrl+c
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	go func() {
		<-c
		sample.Cleanup(sampleName)
		os.Exit(1)
	}()

	resultChan <- CreationResult{State: WillCopy}

	// Create the target folder to copy the sample in to. We do
	// this here in case any of the steps above fail, minimizing
	// the change that we create a dangling empty folder
	targetPath, err := sample.MakeFolder(destination)
	if err != nil {
		resultChan <- CreationResult{Err: err}
		return
	}

	// Perform the copy of the sample given the selected options
	// from the selections above
	err = sample.Copy(targetPath)
	if err != nil {
		resultChan <- CreationResult{Err: err}
		return
	}

	resultChan <- CreationResult{State: DidCopy}

	resultChan <- CreationResult{State: WillConfigure}

	err = sample.ConfigureDotEnv(ctx, targetPath)
	if err != nil {
		resultChan <- CreationResult{Err: err}
		return
	}

	resultChan <- CreationResult{State: DidConfigure}

	resultChan <- CreationResult{State: Done, Path: targetPath, PostInstall: sample.PostInstall()}
}
