package common

type ReturnCode int

const (
	ReturnOK ReturnCode = iota
	ReturnNoConfigFile
	ReturnNoConfigDirectory
	ReturnReadConfigFileFailure
	ReturnCreateDirectoryFailure
	ReturnConfigMarshalFailure
	ReturnConfigUnmarshalFailure
	ReturnConfigWriteFailure
)
