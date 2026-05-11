package twitch

import "time"

type User struct {
	ID              string    `json:"id"`
	Login           string    `json:"login"`
	DisplayName     string    `json:"display_name"`
	Type            string    `json:"type"`
	BroadcasterType string    `json:"broadcaster_type"`
	Description     string    `json:"description"`
	ProfileImageURL string    `json:"profile_image_url"`
	OfflineImageURL string    `json:"offline_image_url"`
	ViewCount       int       `json:"view_count"`
	Email           string    `json:"email"`
	CreatedAt       string    `json:"created_at"`
	PrimaryColorHex string    `json:"primaryColorHex"`
	IsPartner       bool      `json:"isPartner"`
	LastBroadcast   Broadcast `json:"lastBroadcast"`
	Stream          any       `json:"stream"`
	Followers       Followers `json:"followers"`
}

type Stream struct {
	ID           string        `json:"id"`
	UserID       string        `json:"user_id"`
	UserLogin    string        `json:"user_login"`
	UserName     string        `json:"user_name"`
	GameID       string        `json:"game_id"`
	GameName     string        `json:"game_name"`
	Type         string        `json:"type"`
	Title        string        `json:"title"`
	ViewerCount  int           `json:"viewer_count"`
	StartedAt    time.Time     `json:"started_at"`
	Language     string        `json:"language"`
	ThumbnailURL string        `json:"thumbnail_url"`
	TagIds       []interface{} `json:"tag_ids"`
	Tags         []string      `json:"tags"`
	IsMature     bool          `json:"is_mature"`
	CreatedAt    time.Time     `json:"createdAt"`
	Game         Game          `json:"game"`
}

type Channel struct {
	ID                          string   `json:"id"`
	BroadcasterID               string   `json:"broadcaster_id"`
	BroadcasterLogin            string   `json:"broadcaster_login"`
	BroadcasterName             string   `json:"broadcaster_name"`
	BroadcasterLanguage         string   `json:"broadcaster_language"`
	GameID                      string   `json:"game_id"`
	GameName                    string   `json:"game_name"`
	Title                       string   `json:"title"`
	Delay                       int      `json:"delay"`
	Tags                        []string `json:"tags"`
	ContentClassificationLabels []string `json:"content_classification_labels"`
	IsBrandedContent            bool     `json:"is_branded_content"`
}

type HelixErrResponse struct {
	Error   string `json:"error"`
	Status  int    `json:"status"`
	Message string `json:"message"`
}

type Authorization struct {
	IsForbidden         bool   `json:"isForbidden"`
	ForbiddenReasonCode string `json:"forbiddenReasonCode"`
}

type PlaybackAccessToken struct {
	Value         string        `json:"value"`
	Signature     string        `json:"signature"`
	Authorization Authorization `json:"authorization"`
	Typename      string        `json:"__typename"`
}

type VideoQuality struct {
	FrameRate float64 `json:"frameRate"`
	Quality   string  `json:"quality"`
	SourceURL string  `json:"sourceURL"`
}

type Owner struct {
	ID              string `json:"id"`
	DisplayName     string `json:"displayName"`
	Login           string `json:"login"`
	ProfileImageURL string `json:"profileImageURL"`
	PrimaryColorHex string `json:"primaryColorHex"`
	Typename        string `json:"__typename"`
}

type Self struct {
	IsRestricted   bool `json:"isRestricted"`
	ViewingHistory struct {
		Position int `json:"position"`
	} `json:"viewingHistory"`
	IsEditor bool   `json:"isEditor"`
	Typename string `json:"__typename"`
}

type Video struct {
	ID                  string        `json:"id"`
	Title               string        `json:"title"`
	PreviewThumbnailURL string        `json:"previewThumbnailURL"`
	PublishedAt         time.Time     `json:"publishedAt"`
	ViewCount           int64         `json:"viewCount"`
	LengthSeconds       int64         `json:"lengthSeconds"`
	AnimatedPreviewURL  string        `json:"animatedPreviewURL"`
	BroadcastType       string        `json:"broadcastType"`
	ContentTags         []interface{} `json:"contentTags"`
	Self                Self          `json:"self"`
	Game                Game          `json:"game"`
	Owner               Owner         `json:"owner"`
	CreatedAt           time.Time     `json:"createdAt"`
	SeekPreviewsURL     string        `json:"seekPreviewsURL"`
}

type ClipAccessToken struct {
	ID                  string              `json:"id"`
	PlaybackAccessToken PlaybackAccessToken `json:"playbackAccessToken"`
	VideoQualities      []VideoQuality      `json:"videoQualities"`
}

type Curator struct {
	ID              string `json:"id"`
	Login           string `json:"login"`
	DisplayName     string `json:"displayName"`
	ProfileImageURL string `json:"profileImageURL"`
}

type Asset struct {
	ID               string         `json:"id"`
	AspectRatio      float64        `json:"aspectRatio"`
	Type             string         `json:"type"`
	CreatedAt        time.Time      `json:"createdAt"`
	CreationState    string         `json:"creationState"`
	Curator          Curator        `json:"curator"`
	ThumbnailURL     string         `json:"thumbnailURL"`
	VideoQualities   []VideoQuality `json:"videoQualities"`
	PortraitMetadata interface{}    `json:"portraitMetadata"`
}

type Game struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	BoxArtURL   string `json:"boxArtURL"`
	DisplayName string `json:"displayName"`
	Slug        string `json:"slug"`
}

type Broadcast struct {
	ID        string      `json:"id"`
	Title     interface{} `json:"title"`
	StartedAt time.Time   `json:"startedAt"`
	Game      Game        `json:"game"`
}

type Followers struct {
	TotalCount int `json:"totalCount"`
}

type Broadcaster struct {
	ID              string      `json:"id"`
	Login           string      `json:"login"`
	DisplayName     string      `json:"displayName"`
	PrimaryColorHex string      `json:"primaryColorHex"`
	IsPartner       bool        `json:"isPartner"`
	ProfileImageURL string      `json:"profileImageURL"`
	Followers       Followers   `json:"followers"`
	Stream          interface{} `json:"stream"`
	LastBroadcast   Broadcast   `json:"lastBroadcast"`
	Self            Self        `json:"self"`
}

type Clip struct {
	ID                     string              `json:"id"`
	Slug                   string              `json:"slug"`
	URL                    string              `json:"url"`
	EmbedURL               string              `json:"embedURL"`
	Title                  string              `json:"title"`
	ViewCount              int64               `json:"viewCount"`
	Language               string              `json:"language"`
	IsFeatured             bool                `json:"isFeatured"`
	Assets                 []Asset             `json:"assets"`
	Curator                Curator             `json:"curator"`
	Game                   Game                `json:"game"`
	Broadcast              Broadcast           `json:"broadcast"`
	Broadcaster            Broadcaster         `json:"broadcaster"`
	ThumbnailURL           string              `json:"thumbnailURL"`
	CreatedAt              time.Time           `json:"createdAt"`
	IsPublished            bool                `json:"isPublished"`
	DurationSeconds        int                 `json:"durationSeconds"`
	ChampBadge             interface{}         `json:"champBadge"`
	PlaybackAccessToken    PlaybackAccessToken `json:"playbackAccessToken"`
	Video                  Video               `json:"video"`
	VideoOffsetSeconds     int                 `json:"videoOffsetSeconds"`
	VideoQualities         []VideoQuality      `json:"videoQualities"`
	IsViewerEditRestricted bool                `json:"isViewerEditRestricted"`
}

type Commenter struct {
	DisplayName string `json:"displayName"`
	ID          string `json:"id"`
	Login       string `json:"login"`
}

type UserBadge struct {
	ID      string `json:"id"`
	SetID   string `json:"setID"`
	Version string `json:"version"`
}

type Fragments struct {
	Emote interface{} `json:"emote"`
	Text  string      `json:"text"`
}

type Message struct {
	Fragments  []Fragments `json:"fragments"`
	UserBadges []UserBadge `json:"userBadges"`
	UserColor  string      `json:"userColor"`
}

type PageInfo struct {
	HasNextPage     bool `json:"hasNextPage"`
	HasPreviousPage bool `json:"hasPreviousPage"`
}

type Creator struct {
	Channel Channel `json:"channel"`
	ID      string  `json:"id"`
}

type VideoMetadata struct {
	User        User  `json:"user"`
	CurrentUser any   `json:"currentUser"`
	Video       Video `json:"video"`
}

type SubVODResponse struct {
	Video Video `json:"video"`
}

type UseLiveBroadcast struct {
	ID            string    `json:"id"`
	LastBroadcast Broadcast `json:"lastBroadcast"`
}

type BroadcastSettings struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

// GraphQL Responses

type VideoCommentsByOffsetOrCursor struct {
	Data struct {
		Video struct {
			Comments struct {
				Edges []struct {
					Cursor string `json:"cursor"`
					Node   struct {
						ID                   string    `json:"id"`
						Message              Message   `json:"message"`
						Commenter            Commenter `json:"commenter"`
						ContentOffsetSeconds int       `json:"contentOffsetSeconds"`
						CreatedAt            time.Time `json:"createdAt"`
					} `json:"node"`
				} `json:"edges"`
				PageInfo PageInfo `json:"pageInfo"`
			} `json:"comments"`
			Creator Creator `json:"creator"`
			ID      string  `json:"id"`
		} `json:"video"`
	} `json:"data"`
	Extensions struct {
		DurationMilliseconds int    `json:"durationMilliseconds"`
		OperationName        string `json:"operationName"`
		RequestID            string `json:"requestID"`
	} `json:"extensions"`
}

type PlaybackAccessToken_Template struct {
	Data struct {
		PlaybackAccessToken PlaybackAccessToken `json:"streamPlaybackAccessToken"`
	} `json:"data"`
	Extensions struct {
		DurationMilliseconds int    `json:"durationMilliseconds"`
		OperationName        string `json:"operationName"`
		RequestID            string `json:"requestID"`
	} `json:"extensions"`
}

type NielsenContentMetadata struct {
	Data struct {
		Video Video `json:"video"`
	} `json:"data"`
	Extensions struct {
		DurationMilliseconds int    `json:"durationMilliseconds"`
		OperationName        string `json:"operationName"`
		RequestID            string `json:"requestID"`
	} `json:"extensions"`
}

type FilterableVideoTower_Videos struct {
	Data struct {
		User struct {
			ID     string `json:"id"`
			Videos struct {
				Edges []struct {
					Cursor time.Time `json:"cursor"`
					Node   struct {
						AnimatedPreviewURL  string `json:"animatedPreviewURL"`
						Game                Game   `json:"game"`
						BroadcastIdentifier struct {
							ID       string `json:"id"`
							Typename string `json:"__typename"`
						} `json:"broadcastIdentifier"`
						ID            string `json:"id"`
						LengthSeconds int    `json:"lengthSeconds"`
						Owner         struct {
							DisplayName     string      `json:"displayName"`
							ID              string      `json:"id"`
							Login           string      `json:"login"`
							ProfileImageURL string      `json:"profileImageURL"`
							PrimaryColorHex interface{} `json:"primaryColorHex"`
							Roles           struct {
								IsPartner bool   `json:"isPartner"`
								Typename  string `json:"__typename"`
							} `json:"roles"`
							Typename string `json:"__typename"`
						} `json:"owner"`
						PreviewThumbnailURL string        `json:"previewThumbnailURL"`
						PublishedAt         time.Time     `json:"publishedAt"`
						Self                Self          `json:"self"`
						Title               string        `json:"title"`
						ViewCount           int           `json:"viewCount"`
						ResourceRestriction interface{}   `json:"resourceRestriction"`
						ContentTags         []interface{} `json:"contentTags"`
						Typename            string        `json:"__typename"`
					} `json:"node"`
					Typename string `json:"__typename"`
				} `json:"edges"`
				PageInfo struct {
					HasNextPage bool   `json:"hasNextPage"`
					Typename    string `json:"__typename"`
				} `json:"pageInfo"`
				Typename string `json:"__typename"`
			} `json:"videos"`
			Typename string `json:"__typename"`
		} `json:"user"`
	} `json:"data"`
	Extensions struct {
		DurationMilliseconds int    `json:"durationMilliseconds"`
		OperationName        string `json:"operationName"`
		RequestID            string `json:"requestID"`
	} `json:"extensions"`
}
