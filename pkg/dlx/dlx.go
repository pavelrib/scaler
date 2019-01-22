package dlx

import (
	"net/http"

	"github.com/v3io/scaler/pkg"

	"github.com/nuclio/errors"
	"github.com/nuclio/logger"
)

type DLX struct {
	logger        logger.Logger
	listenAddress string
	handler       Handler
}

func NewDLX(logger logger.Logger,
	resourceScaler scaler.ResourceScaler,
	options scaler.DLXOptions) (*DLX, error) {
	resourceStarter, err := NewResourceStarter(logger, resourceScaler, options.Namespace)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create function starter")
	}

	handler, err := NewHandler(logger,
		resourceStarter,
		options.TargetNameHeader,
		options.TargetPathHeader,
		options.TargetPort)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create handler")
	}

	return &DLX{
		logger:        logger,
		listenAddress: options.ListenAddress,
		handler:       handler,
	}, nil
}

func (d *DLX) Start() error {
	d.logger.InfoWith("Starting",
		"listenAddress", d.listenAddress)

	http.HandleFunc("/", d.handler.HandleFunc)
	if err := http.ListenAndServe(d.listenAddress, nil); err != nil {
		return errors.Wrap(err, "Failed to serve dlx service")
	}
	return nil
}
