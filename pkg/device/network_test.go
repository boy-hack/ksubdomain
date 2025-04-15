package device

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestAutoGetDevices(t *testing.T) {
	ether, err := AutoGetDevices([]string{"1.1.1.1"})
	assert.NoError(t, err)
	PrintDeviceInfo(ether)
}
