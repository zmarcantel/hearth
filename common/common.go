package common

type ReturnCode int

const (
	ReturnOK ReturnCode = iota
	ReturnNoConfigFile
	ReturnNoConfigDirectory
	ReturnConfigReadFileFailure
	ReturnCreateDirectoryFailure
	ReturnConfigMarshalFailure
	ReturnConfigUnmarshalFailure
	ReturnConfigWriteFailure
	ReturnRepoCreateFailure
)
