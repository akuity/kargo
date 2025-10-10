package certloader

import (
	"crypto/tls"
	"sync"

	"github.com/akuity/kargo/internal/server/certwatcher"
	"github.com/akuity/kargo/pkg/logging"
)

type CertLoader struct {
	logger            *logging.Logger
	certPath, keyPath string
	done              chan struct{}
	certWatcher       *certwatcher.CertWatcher

	cert     *tls.Certificate
	certLock sync.RWMutex
}

func NewCertLoader(logger *logging.Logger, certPath, keyPath string) (*CertLoader, error) {
	certWatcher, err := certwatcher.NewCertWatcher(certPath, keyPath)
	if err != nil {
		return nil, err
	}

	cert, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		return nil, err
	}

	c := &CertLoader{
		logger:      logger,
		certWatcher: certWatcher,
		cert:        &cert,
	}

	go c.run()

	return c, nil
}

func (c *CertLoader) GetCertificate(_ *tls.ClientHelloInfo) (*tls.Certificate, error) {
	c.certLock.RLock()
	defer c.certLock.RUnlock()
	return c.cert, nil
}

func (c *CertLoader) Close() {
	c.certWatcher.Close()
	close(c.done)
}

func (c *CertLoader) run() {
	go c.certWatcher.Run()
	for {
		select {
		case <-c.done:
			return
		case _, ok := <-c.certWatcher.Events():
			if !ok {
				return
			}
			cert, err := tls.LoadX509KeyPair(c.certPath, c.keyPath)
			if err != nil {
				c.logger.Error(err, "failed to load certificate and key pair, keeping existing certificate")
				continue
			}
			c.certLock.Lock()
			c.cert = &cert
			c.certLock.Unlock()
		}
	}
}
