package m3u8

type VariantPlaylist struct {
	Bandwidth  string `json:"BANDWIDTH"`
	Codecs     string `json:"CODECS"`
	Resolution string `json:"RESOLUTION"`
	Video      string `json:"VIDEO"`
	FrameRate  string `json:"FRAME-RATE"`
	URL        string
	Serialized string
}

type MasterPlaylist struct {
	Origin          string `json:"ORIGIN"`
	B               bool   `json:"B"`
	Region          string `json:"REGION"`
	UserIP          string `json:"USER-IP"`
	ServingID       string `json:"SERVING-ID"`
	Cluster         string `json:"CLUSTER"`
	UserCountry     string `json:"USER-COUNTRY"`
	ManifestCluster string `json:"MANIFEST-CLUSTER"`
	UsherURL        string
	Lists           []VariantPlaylist
	Serialized      string
}
