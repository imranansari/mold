package main

// Artifacts contains docker images to be built and optionally publish them
type Artifacts struct {
	Registry string `yaml:",omitempty"` // default registry value
	Images   []ImageConfig
	Publish  []string // branch/tag's to publish images on

	S3 []S3Config
}

// ValidateImageConfigs validates all image configs
func (art *Artifacts) ValidateImageConfigs() error {
	for _, v := range art.Images {
		if err := v.Validate(); err != nil {
			return err
		}
	}

	for _, v := range art.S3 {
		if err := v.Validate(); err != nil {
			return err
		}
	}

	return nil
}

// GetImage return an image config of the name.  If the image is not found, nil
// is returned
func (art *Artifacts) GetImage(name string) *ImageConfig {
	for i, v := range art.Images {
		if v.Name == name {
			return &art.Images[i]
		}
	}
	return nil
}

// setDefaults sets all the registry values by setting the default
// registry value if one is not specified along with setting the default
// docker file path
func (art *Artifacts) setDefaults() {
	for i, img := range art.Images {
		if len(img.Dockerfile) == 0 {
			art.Images[i].Dockerfile = "Dockerfile"
		}
	}

	if len(art.Registry) > 0 {
		for i, img := range art.Images {
			if len(img.Registry) == 0 {
				art.Images[i].Registry = art.Registry
			}
		}
	}

}

// loadEnv loads env variables for s3 artifacts
func (art *Artifacts) loadEnv() {
	for _, s3c := range art.S3 {
		s3c.LoadCredentialsFromEnv()
	}
}
