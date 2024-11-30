package flowcontrol

import (
	"testing"
	"time"

	"github.com/quic-go/quic-go/internal/protocol"
	"github.com/quic-go/quic-go/internal/qerr"
	"github.com/quic-go/quic-go/internal/utils"

	"github.com/stretchr/testify/require"
)

func TestConnectionFlowControlWindowUpdate(t *testing.T) {
	fc := NewConnectionFlowController(
		100, // initial receive window
		100, // max receive window
		nil,
		&utils.RTTStats{},
		utils.DefaultLogger,
	)
	require.Zero(t, fc.GetWindowUpdate(time.Now()))
	fc.AddBytesRead(100)
	require.Equal(t, protocol.ByteCount(200), fc.GetWindowUpdate(time.Now()))
}

func TestConnectionWindowAutoTuningNotAllowed(t *testing.T) {
	// the RTT is 1 second
	rttStats := &utils.RTTStats{}
	rttStats.UpdateRTT(time.Second, 0, time.Now())
	require.Equal(t, time.Second, rttStats.SmoothedRTT())

	callbackCalledWith := protocol.InvalidByteCount
	fc := NewConnectionFlowController(
		100, // initial receive window
		150, // max receive window
		func(size protocol.ByteCount) bool {
			callbackCalledWith = size
			return false
		},
		rttStats,
		utils.DefaultLogger,
	)
	now := time.Now()
	require.NoError(t, fc.IncrementHighestReceived(100, now))
	fc.AddBytesRead(90)
	require.Equal(t, protocol.InvalidByteCount, callbackCalledWith)
	require.Equal(t, protocol.ByteCount(90+100), fc.GetWindowUpdate(now.Add(time.Millisecond)))
	require.Equal(t, protocol.ByteCount(150-100), callbackCalledWith)
}

func TestConnectionFlowControlViolation(t *testing.T) {
	fc := NewConnectionFlowController(100, 100, nil, &utils.RTTStats{}, utils.DefaultLogger)
	require.NoError(t, fc.IncrementHighestReceived(40, time.Now()))
	require.NoError(t, fc.IncrementHighestReceived(60, time.Now()))
	err := fc.IncrementHighestReceived(1, time.Now())
	var terr *qerr.TransportError
	require.ErrorAs(t, err, &terr)
	require.Equal(t, qerr.FlowControlError, terr.ErrorCode)
}

// TODO (#4732): add a test for successfully resetting the flow controller
func TestConnectionFlowControllerReset(t *testing.T) {
	fc := NewConnectionFlowController(0, 0, nil, &utils.RTTStats{}, utils.DefaultLogger)
	fc.AddBytesRead(1)
	require.EqualError(t, fc.Reset(), "flow controller reset after reading data")
}
