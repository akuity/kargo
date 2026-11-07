package nats

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

type testKey struct {
	Project string
	Name    string
}

type testRow struct {
	Name  string
	Phase string
}

// recordingPublisher keeps every message it is given.
type recordingPublisher struct {
	subjects []string
	data     [][]byte
	err      error
}

func (p *recordingPublisher) Publish(subject string, data []byte) error {
	if p.err != nil {
		return p.err
	}
	p.subjects = append(p.subjects, subject)
	p.data = append(p.data, data)
	return nil
}

func TestTopicPublish(t *testing.T) {
	t.Parallel()
	topic := NewTopic[testKey, testRow]("kargo.tests")
	key := testKey{Project: "p", Name: "r"}
	old := testRow{Name: "r", Phase: "Pending"}
	updated := testRow{Name: "r", Phase: "Running"}

	testCases := []struct {
		name    string
		topic   Topic[testKey, testRow]
		pub     *recordingPublisher
		publish func(Topic[testKey, testRow], Publisher) error
		assert  func(*testing.T, *recordingPublisher, error)
	}{
		{
			name:  "created",
			topic: topic,
			pub:   &recordingPublisher{},
			publish: func(topic Topic[testKey, testRow], pub Publisher) error {
				return topic.PublishCreated(pub, key, updated)
			},
			assert: func(t *testing.T, pub *recordingPublisher, err error) {
				require.NoError(t, err)
				require.Equal(t, []string{"kargo.tests.created"}, pub.subjects)
				e := decode(t, pub.data[0])
				require.Equal(t, Created, e.Kind)
				require.Equal(t, key, e.Key)
				require.Nil(t, e.Old)
				require.Equal(t, &updated, e.New)
			},
		},
		{
			name:  "updated",
			topic: topic,
			pub:   &recordingPublisher{},
			publish: func(topic Topic[testKey, testRow], pub Publisher) error {
				return topic.PublishUpdated(pub, key, old, updated)
			},
			assert: func(t *testing.T, pub *recordingPublisher, err error) {
				require.NoError(t, err)
				require.Equal(t, []string{"kargo.tests.updated"}, pub.subjects)
				e := decode(t, pub.data[0])
				require.Equal(t, Updated, e.Kind)
				require.Equal(t, key, e.Key)
				require.Equal(t, &old, e.Old)
				require.Equal(t, &updated, e.New)
			},
		},
		{
			name:  "deleted",
			topic: topic,
			pub:   &recordingPublisher{},
			publish: func(topic Topic[testKey, testRow], pub Publisher) error {
				return topic.PublishDeleted(pub, key, old)
			},
			assert: func(t *testing.T, pub *recordingPublisher, err error) {
				require.NoError(t, err)
				require.Equal(t, []string{"kargo.tests.deleted"}, pub.subjects)
				e := decode(t, pub.data[0])
				require.Equal(t, Deleted, e.Kind)
				require.Equal(t, key, e.Key)
				require.Equal(t, &old, e.Old)
				require.Nil(t, e.New)
			},
		},
		{
			name:  "a nil publisher publishes nothing",
			topic: topic,
			publish: func(topic Topic[testKey, testRow], _ Publisher) error {
				return topic.PublishCreated(nil, key, updated)
			},
			assert: func(t *testing.T, _ *recordingPublisher, err error) {
				require.NoError(t, err)
			},
		},
		{
			name:  "a topic without a prefix cannot publish",
			topic: Topic[testKey, testRow]{},
			pub:   &recordingPublisher{},
			publish: func(topic Topic[testKey, testRow], pub Publisher) error {
				return topic.PublishCreated(pub, key, updated)
			},
			assert: func(t *testing.T, pub *recordingPublisher, err error) {
				require.ErrorContains(t, err, "no prefix")
				require.Empty(t, pub.subjects)
			},
		},
		{
			name:  "a publisher failure is returned with the subject",
			topic: topic,
			pub:   &recordingPublisher{err: errors.New("disconnected")},
			publish: func(topic Topic[testKey, testRow], pub Publisher) error {
				return topic.PublishUpdated(pub, key, old, updated)
			},
			assert: func(t *testing.T, _ *recordingPublisher, err error) {
				require.ErrorContains(t, err, `"kargo.tests.updated"`)
				require.ErrorContains(t, err, "disconnected")
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			var pub Publisher
			if testCase.pub != nil {
				pub = testCase.pub
			}
			err := testCase.publish(testCase.topic, pub)
			testCase.assert(t, testCase.pub, err)
		})
	}
}

// TestTopicPublishRoundTrip checks that what a topic publishes is what a
// source subscribed to it would accept.
func TestTopicPublishRoundTrip(t *testing.T) {
	t.Parallel()
	topic := NewTopic[testKey, testRow]("kargo.tests")
	pub := &recordingPublisher{}
	require.NoError(t, topic.PublishUpdated(
		pub,
		testKey{Project: "p", Name: "r"},
		testRow{Phase: "Pending"},
		testRow{Phase: "Running"},
	))
	e := decode(t, pub.data[0])
	require.NoError(t, e.Validate())
	require.Equal(t, topic.subjectFor(e.Kind), pub.subjects[0])
}

func decode(t *testing.T, data []byte) Event[testKey, testRow] {
	t.Helper()
	var e Event[testKey, testRow]
	require.NoError(t, json.Unmarshal(data, &e))
	return e
}
