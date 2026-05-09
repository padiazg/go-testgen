package analyzer

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoad_ChannelReturn(t *testing.T) {
	info, err := Load("github.com/padiazg/go-testgen/internal/testdata/sample", "ChannelRecvReturn")
	assert.NoError(t, err)
	assert.NotNil(t, info)

	assert.Equal(t, "ChannelRecvReturn", info.Name)
	assert.Len(t, info.Results, 1)

	r := info.Results[0]
	assert.True(t, r.IsChannel, "expected IsChannel=true")
	assert.Equal(t, 2, r.ChanDir, "expected ChanDir=2 (recv-only)")
	assert.Contains(t, r.TypeName, "<-chan *")
}

func TestLoad_ChannelSendParam(t *testing.T) {
	info, err := Load("github.com/padiazg/go-testgen/internal/testdata/sample", "ChannelSendParam")
	assert.NoError(t, err)
	assert.NotNil(t, info)

	assert.Len(t, info.Params, 1)

	p := info.Params[0]
	assert.True(t, p.IsChannel, "expected IsChannel=true")
	assert.Equal(t, 1, p.ChanDir, "expected ChanDir=1 (send-only)")
	assert.Equal(t, "chan<- string", p.TypeName)
}

func TestLoad_ChannelBidi(t *testing.T) {
	info, err := Load("github.com/padiazg/go-testgen/internal/testdata/sample", "ChannelBidi")
	assert.NoError(t, err)
	assert.NotNil(t, info)

	assert.Len(t, info.Params, 1)
	assert.Len(t, info.Results, 1)

	p := info.Params[0]
	assert.True(t, p.IsChannel)
	assert.Equal(t, 0, p.ChanDir, "expected ChanDir=0 (bidi)")

	r := info.Results[0]
	assert.True(t, r.IsChannel)
	assert.Equal(t, 0, r.ChanDir)
}

func TestLoad_ChannelSlice(t *testing.T) {
	info, err := Load("github.com/padiazg/go-testgen/internal/testdata/sample", "ChannelSlice")
	assert.NoError(t, err)
	assert.NotNil(t, info)

	assert.Len(t, info.Results, 1)
	r := info.Results[0]
	assert.True(t, r.IsChannel)
	assert.Equal(t, 2, r.ChanDir)
	assert.Equal(t, "<-chan []string", r.TypeName)
}

func TestLoad_ChannelPointer(t *testing.T) {
	info, err := Load("github.com/padiazg/go-testgen/internal/testdata/sample", "ChannelPointer")
	assert.NoError(t, err)
	assert.NotNil(t, info)

	assert.Len(t, info.Results, 1)
	r := info.Results[0]
	assert.True(t, r.IsChannel)
	assert.Contains(t, r.TypeName, "*SomeType")
}

func TestLoad_MultiChannelParam(t *testing.T) {
	info, err := Load("github.com/padiazg/go-testgen/internal/testdata/sample", "MultiChannelParam")
	assert.NoError(t, err)
	assert.NotNil(t, info)

	assert.Len(t, info.Params, 3)

	p0 := info.Params[0]
	assert.True(t, p0.IsChannel)
	assert.Equal(t, 1, p0.ChanDir, "chan<- int is send-only")

	p1 := info.Params[1]
	assert.True(t, p1.IsChannel)
	assert.Equal(t, 0, p1.ChanDir, "chan int is bidi")

	p2 := info.Params[2]
	assert.True(t, p2.IsChannel)
	assert.Equal(t, 2, p2.ChanDir, "<-chan string is recv-only")
}

func TestLoad_PackageHasNoErrors(t *testing.T) {
	info, err := Load("github.com/padiazg/go-testgen/internal/testdata/sample", "ChannelRecvReturn")
	assert.NoError(t, err)
	assert.NotNil(t, info)
}
