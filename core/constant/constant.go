package constant

const (
	TRIGGER_TYPE      = "signed-tag"
	TRIGGER_BRANCH    = "master"
	GPG_REPO_GPG_PATH = "keys/gpg"
	GPG_REPO_ASC_PATH = "keys/asc"
	GPG_REPO_BRANCH   = "master"
	REDACTION         = "(X_REDACTION_X)"
	SIGNED_TAG_VALUE  = "signed-tag"
	TAG_VALUE         = "tag"
	BRANCH_VALUE      = "branch"
	TAG_REF_PREFIX    = "refs/tags/"
)

const (
	NOT_APPLICABLE int = iota - 1
	CONTINUE
	SILENTLY
	ERROR
)
