package deployer

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestVolcDeployer_Deploy(t *testing.T) {
	v, err := CreateVolcDeployer()
	if err != nil {
		t.Error(err)
		return
	}

	err, _ = v.listCertBind()
	if err != nil {
		t.Error(err)
		return
	}
}

func TestMatchDomain(t *testing.T) {
	assert.Equal(t, true, matchDomain(
		[]string{"*.example.com"},
		[]string{"a.example.com"}))
	assert.Equal(t, false, matchDomain(
		[]string{"*.example.com"},
		[]string{"foo.bar.example.com"}))
	assert.Equal(t, true, matchDomain(
		[]string{"*.foo.com", "*.bar.com"},
		[]string{"foo.bar.com"}))
	assert.Equal(t, true, matchDomain(
		[]string{"*.foo.com", "*.bar.com", "bar.com"},
		[]string{"foo.bar.com", "bar.foo.com", "bar.com"}))
	assert.Equal(t, false, matchDomain(
		[]string{"*.foo.com", "*.bar.com", "bar.com"},
		[]string{"foo.bar.com", "bar.foo.com", "foo.com"}))
}
