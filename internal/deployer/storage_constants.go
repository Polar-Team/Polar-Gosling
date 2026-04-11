package deployer

// Static folder prefix constants for the unified S3 storage bucket.
// These are hardcoded values and MUST NOT be made configurable.
// The bucket name itself is configured via MOTHERGOOSE_S3_BUCKET.
const (
	StoragePrefixBinaries     = "binaries/"
	StoragePrefixStates       = "states/"
	StoragePrefixPluginCache  = "plugin-cache/"
	StoragePrefixRunnersCache = "runners-cache/"
)
