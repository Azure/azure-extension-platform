package environmentmanager

import (
	"github.com/Azure/azure-extension-platform/pkg/handlerenv"
	"github.com/Azure/azure-extension-platform/pkg/seqno"
	"github.com/Azure/azure-extension-platform/pkg/settings"
	"github.com/go-kit/kit/log"
)

// Allows for mocking all environment operations when running tests against VM extensions
type IGetVMExtensionEnvironmentManager interface {
	GetHandlerEnvironment(name string, version string) (*handlerenv.HandlerEnvironment, error)
	FindSeqNum(ctx log.Logger, configFolder string) (uint, error)
	GetCurrentSequenceNumber(ctx log.Logger, retriever seqno.ISequenceNumberRetriever, name, version string) (uint, error)
	GetHandlerSettings(ctx log.Logger, he *handlerenv.HandlerEnvironment, seqNo uint) (*settings.HandlerSettings, error)
	SetSequenceNumberInternal(extensionName, extensionVersion string, seqNo uint) error
}
