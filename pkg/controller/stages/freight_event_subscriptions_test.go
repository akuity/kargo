package stages

import (
	"context"
	"testing"
	"time"

	natstest "github.com/nats-io/nats-server/v2/test"
	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	kargoEvent "github.com/akuity/kargo/pkg/event"
	natsevent "github.com/akuity/kargo/pkg/event/nats"
)

func testFreightForEvents() *kargoapi.Freight {
	return &kargoapi.Freight{
		ObjectMeta: metav1.ObjectMeta{Namespace: "proj", Name: "freight-1"},
		Origin: kargoapi.FreightOrigin{
			Kind: kargoapi.FreightOriginKindWarehouse,
			Name: "wh1",
		},
	}
}

func Test_stages_subscribeFreightCreated(t *testing.T) {
	srv := natstest.RunRandClientPortServer()
	t.Cleanup(srv.Shutdown)

	conn, err := nats.Connect(srv.ClientURL())
	require.NoError(t, err)
	t.Cleanup(conn.Close)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch, err := subscribeFreightCreated(ctx, conn)
	require.NoError(t, err)

	evt, err := kargoEvent.NewFreightCreated("created", "test", testFreightForEvents())
	require.NoError(t, err)

	sender := natsevent.NewDefaultingEventSender(conn, kargoEvent.EventsSubjectPrefix, "test")
	require.NoError(t, sender.Send(ctx, evt))
	sender.Shutdown()

	select {
	case ge := <-ch:
		require.NotNil(t, ge.Object)
		require.Equal(t, "proj", ge.Object.Namespace)
		require.Equal(t, "freight-1", ge.Object.Name)
		require.Equal(t, "wh1", ge.Object.Origin.Name)
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for a GenericEvent from the FreightCreated subscription")
	}
}

func Test_stages_subscribeFreightVerifiedInStage(t *testing.T) {
	srv := natstest.RunRandClientPortServer()
	t.Cleanup(srv.Shutdown)

	conn, err := nats.Connect(srv.ClientURL())
	require.NoError(t, err)
	t.Cleanup(conn.Close)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch, err := subscribeFreightVerifiedInStage(ctx, conn)
	require.NoError(t, err)

	evt, err := kargoEvent.NewFreightVerifiedInStage("test", "actor", "up", testFreightForEvents())
	require.NoError(t, err)

	sender := natsevent.NewDefaultingEventSender(conn, kargoEvent.EventsSubjectPrefix, "test")
	require.NoError(t, sender.Send(ctx, evt))
	sender.Shutdown()

	select {
	case ge := <-ch:
		require.NotNil(t, ge.Object)
		require.Equal(t, "proj", ge.Object.Namespace)
		require.Equal(t, "freight-1", ge.Object.Name)
		require.Contains(t, ge.Object.Status.VerifiedIn, "up")
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for a GenericEvent from the FreightVerifiedInStage subscription")
	}
}

func Test_stages_subscribeFreightCurrentlyInStage(t *testing.T) {
	srv := natstest.RunRandClientPortServer()
	t.Cleanup(srv.Shutdown)

	conn, err := nats.Connect(srv.ClientURL())
	require.NoError(t, err)
	t.Cleanup(conn.Close)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch, err := subscribeFreightCurrentlyInStage(ctx, conn)
	require.NoError(t, err)

	sender := natsevent.NewDefaultingEventSender(conn, kargoEvent.EventsSubjectPrefix, "test")

	// Verified: the flag is encoded into Status.VerifiedIn.
	verifiedEvt, err := kargoEvent.NewFreightCurrentlyInStage("test", "actor", "up", true, testFreightForEvents())
	require.NoError(t, err)
	require.NoError(t, sender.Send(ctx, verifiedEvt))

	select {
	case ge := <-ch:
		require.NotNil(t, ge.Object)
		require.Equal(t, "proj", ge.Object.Namespace)
		require.Equal(t, "freight-1", ge.Object.Name)
		require.Contains(t, ge.Object.Status.CurrentlyIn, "up")
		require.Contains(t, ge.Object.Status.VerifiedIn, "up")
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for a verified GenericEvent")
	}

	// Unverified: current in the Stage but not verified there.
	unverifiedEvt, err := kargoEvent.NewFreightCurrentlyInStage("test", "actor", "up", false, testFreightForEvents())
	require.NoError(t, err)
	require.NoError(t, sender.Send(ctx, unverifiedEvt))
	sender.Shutdown()

	select {
	case ge := <-ch:
		require.NotNil(t, ge.Object)
		require.Contains(t, ge.Object.Status.CurrentlyIn, "up")
		require.NotContains(t, ge.Object.Status.VerifiedIn, "up")
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for an unverified GenericEvent")
	}
}

func Test_stages_subscribeFreightApproved(t *testing.T) {
	srv := natstest.RunRandClientPortServer()
	t.Cleanup(srv.Shutdown)

	conn, err := nats.Connect(srv.ClientURL())
	require.NoError(t, err)
	t.Cleanup(conn.Close)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch, err := subscribeFreightApproved(ctx, conn)
	require.NoError(t, err)

	evt, err := kargoEvent.NewFreightApproved("approved", "actor", "stage-a", testFreightForEvents())
	require.NoError(t, err)

	sender := natsevent.NewDefaultingEventSender(conn, kargoEvent.EventsSubjectPrefix, "test")
	require.NoError(t, sender.Send(ctx, evt))
	sender.Shutdown()

	select {
	case ge := <-ch:
		require.NotNil(t, ge.Object)
		require.Equal(t, "proj", ge.Object.Namespace)
		require.Equal(t, "freight-1", ge.Object.Name)
		// The Stage the Freight was approved for is carried in Status.ApprovedFor.
		require.Contains(t, ge.Object.Status.ApprovedFor, "stage-a")
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for a GenericEvent from the FreightApproved subscription")
	}
}

func Test_stages_subscribeFreightCreated_dropsUndecodableMessages(t *testing.T) {
	srv := natstest.RunRandClientPortServer()
	t.Cleanup(srv.Shutdown)

	conn, err := nats.Connect(srv.ClientURL())
	require.NoError(t, err)
	t.Cleanup(conn.Close)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch, err := subscribeFreightCreated(ctx, conn)
	require.NoError(t, err)

	subject, err := kargoEvent.NewEventSubject(string(kargoapi.EventTypeFreightCreated))
	require.NoError(t, err)

	require.NoError(t, conn.Publish(subject, []byte("not a cloudevent")))
	require.NoError(t, conn.Flush())

	select {
	case ge := <-ch:
		t.Fatalf("expected no event, got %+v", ge.Object)
	case <-time.After(200 * time.Millisecond):
		// Expected: nothing forwarded.
	}
}
