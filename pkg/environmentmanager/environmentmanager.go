// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
package environmentmanager

import (
	"github.com/Azure/azure-extension-platform/pkg/handlerenv"
	"github.com/Azure/azure-extension-platform/pkg/logging"
	"github.com/Azure/azure-extension-platform/pkg/seqno"
	"github.com/Azure/azure-extension-platform/pkg/settings"
)

// Allows for mocking all environment operations when running tests against VM extensions
type IGetVMExtensionEnvironmentManager interface {
	GetHandlerEnvironment(name string, version string) (*handlerenv.HandlerEnvironment, error)
	FindSeqNum(el logging.ILogger, configFolder string) (uint, error)
	GetCurrentSequenceNumber(el logging.ILogger, retriever seqno.ISequenceNumberRetriever, name, version string) (uint, error)
	GetHandlerSettings(el logging.ILogger, he *handlerenv.HandlerEnvironment) (*settings.HandlerSettings, error)
	SetSequenceNumberInternal(extensionName, extensionVersion string, seqNo uint) error
}
