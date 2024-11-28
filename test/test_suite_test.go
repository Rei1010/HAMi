package e2e

import (
	"testing"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"

	"gitlab.daocloud.cn/ndx/baize/test/utils"
)

func init() {
	testing.Init()
}

func TestInit(t *testing.T) {
	gomega.RegisterFailHandler(ginkgo.Fail)
	ginkgo.RunSpecs(t, "Test HAMi Suite")
}

var _ = ginkgo.BeforeSuite(func() {
	utils.NewClient(utils.KubeConfigPath, utils.DefaultURL)
})
