module git.enumerated.dev/bubble-package-manager/bpm-utils/src/bpm-package

go 1.23

require (
	bpm-utils-shared v1.0.0
	github.com/spf13/pflag v1.0.10
	gopkg.in/yaml.v3 v3.0.1
)

require github.com/drone/envsubst v1.0.3 // indirect

replace bpm-utils-shared => ../bpm-utils-shared
