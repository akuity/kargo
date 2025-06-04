package external

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

// WebhookReceiver is an interface for components that handle inbound webhooks.
type WebhookReceiver interface {
	// getReceiverType returns the type of this receiver.
	getReceiverType() string
	// getSecretName returns the name of the Secret upon which this receiver
	// relies.
	getSecretName() string
	// getSecretValues extracts a list of receiver-specific values from the
	// provided Secret data.
	getSecretValues(map[string][]byte) ([]string, error)
	// setSecretData sets the Secret data for this receiver. This is used to
	// later when handling inbound webhooks.
	setSecretData(map[string][]byte)
	// setDetails sets the details of the WebhookReceiver in the form of
	// kargoapi.WebhookReceiverDetails.
	setDetails(kargoapi.WebhookReceiverDetails)
	// GetDetails returns the details of the WebhookReceiver in the form of
	// kargoapi.WebhookReceiverDetails.
	GetDetails() kargoapi.WebhookReceiverDetails
	// GetHandler returns an http.HandlerFunc for handling inbound webhook
	// requests.
	GetHandler() http.HandlerFunc
}

// baseWebhookReceiver is a base implementation of WebhookReceiver that provides
// common functionality for all WebhookReceiver implementations. It is not
// intended to be used directly.
type baseWebhookReceiver struct {
	client     client.Client
	project    string
	secretName string
	secretData map[string][]byte
	details    kargoapi.WebhookReceiverDetails
}

// getSecretName implements WebhookReceiver.
func (b *baseWebhookReceiver) getSecretName() string {
	return b.secretName
}

// setSecretData implements WebhookReceiver.
func (b *baseWebhookReceiver) setSecretData(secretData map[string][]byte) {
	b.secretData = secretData
}

// setDetails implements WebhookReceiver.
func (b *baseWebhookReceiver) setDetails(
	details kargoapi.WebhookReceiverDetails,
) {
	b.details = details
}

// GetDetails implements WebhookReceiver.
func (b *baseWebhookReceiver) GetDetails() kargoapi.WebhookReceiverDetails {
	return b.details
}

// NewReceiver returns an appropriate implementation of WebhookReceiver based on
// the provided kargoapi.WebhookReceiverConfig.
func NewReceiver(
	ctx context.Context,
	c client.Client,
	baseURL string,
	project string,
	secretsNamespace string,
	cfg kargoapi.WebhookReceiverConfig,
) (WebhookReceiver, error) {
	// Pick an appropriate WebhookReceiver implementation based on the
	// configuration provided.
	var receiver WebhookReceiver
	switch {
	case cfg.GitHub != nil:
		receiver = newGitHubWebhookReceiver(c, project, cfg)
	default:
		return nil, errors.New(
			"WebhookReceiverConfig has no configuration for a known receiver type",
		)
	}
	secretName := receiver.getSecretName()
	secret := &corev1.Secret{}
	if err := c.Get(
		ctx,
		client.ObjectKey{
			Namespace: secretsNamespace,
			Name:      secretName,
		},
		secret,
	); err != nil {
		return nil, fmt.Errorf(
			"error getting Secret %q in namespace %q: %w",
			secretName, secretsNamespace, err,
		)
	}
	// The receiver is likely to rely on the Secret data when handling inbound
	// webhooks.
	receiver.setSecretData(secret.Data)
	// Extract the values of select keys from the Secret data for use in building
	// the details of the WebhookReceiver, namely the URL path and URL.
	secretValues, err := receiver.getSecretValues(secret.Data)
	if err != nil {
		return nil, fmt.Errorf(
			"error extracting secret values from Secret %q in namespace %q: %w",
			secretName, secretsNamespace, err,
		)
	}
	// Build the details of the WebhookReceiver in the form of
	// kargoapi.WebhookReceiverDetails.
	details, err := getDetails(
		baseURL,
		project,
		receiver.getReceiverType(),
		cfg.Name,
		secretValues...,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"error building details for WebhookReceiver %q: %w",
			cfg.Name,
			err,
		)
	}
	// Make sure the details are retrievable from the WebhookReceiver later.
	receiver.setDetails(details)
	return receiver, nil
}

// getDetails is a utility function that builds the details of a WebhookReceiver
// in the form of kargoapi.WebhookReceiverDetails.
func getDetails(
	baseURL string,
	project string,
	receiverType string,
	receiverName string,
	secretValues ...string,
) (kargoapi.WebhookReceiverDetails, error) {
	receiverPath := buildWebhookPath(project, receiverType, receiverName, secretValues...)
	receiverURL, err := url.JoinPath(baseURL, receiverPath)
	if err != nil {
		return kargoapi.WebhookReceiverDetails{}, err
	}
	return kargoapi.WebhookReceiverDetails{
		Name: receiverName,
		Path: receiverPath,
		URL:  receiverURL,
	}, nil
}

// buildWebhookPath generates a unique path for inbound webhooks based on the
// provided Project name, receiver type, receiver name, and secret values. The
// path is formatted as "/webhook/{receiverType}/{hash}" where the hash is a
// SHA-256 hash of the concatenated project name, receiver name, and secret
// values.
func buildWebhookPath(
	project string,
	receiverType string,
	receiverName string,
	secretValues ...string,
) string {
	// Warning: Changes to this line could alter URLs that users may already be
	// using!!!
	input := []byte(project + receiverName + strings.Join(secretValues, ""))
	h := sha256.New()
	_, _ = h.Write(input)
	return fmt.Sprintf("/webhook/%s/%x", receiverType, h.Sum(nil))
}
