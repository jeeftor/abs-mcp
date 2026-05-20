package abs

// JSONValue is an arbitrary JSON value returned by ABS for endpoints whose
// response shape is broad or source-version dependent.
type JSONValue any

// User is the authenticated Audiobookshelf user returned by /api/me.
type User struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Type     string `json:"type"`
	IsActive bool   `json:"isActive"`
}

// Library represents an Audiobookshelf library.
type Library struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	MediaType    string   `json:"mediaType"`
	Folders      []Folder `json:"folders"`
	DisplayOrder int      `json:"displayOrder"`
	Icon         string   `json:"icon"`
	CreatedAt    int64    `json:"createdAt"`
	LastUpdate   int64    `json:"lastUpdate"`
}

// Folder represents a filesystem folder mounted into a library.
type Folder struct {
	ID        string `json:"id"`
	Path      string `json:"path"`
	FullPath  string `json:"fullPath"`
	LibraryID string `json:"libraryId,omitempty"`
}

// LibraryItemsResponse is the paginated response for library item listing.
type LibraryItemsResponse struct {
	Results []LibraryItem `json:"results"`
	Total   int           `json:"total"`
	Limit   int           `json:"limit"`
	Page    int           `json:"page"`
	Offset  int           `json:"offset"`
}

// LibraryItemsOptions selects one ABS library items page.
type LibraryItemsOptions struct {
	Limit          int
	Page           int
	Sort           string
	Desc           bool
	Filter         string
	Include        []string
	Minified       bool
	CollapseSeries bool
}

// ScanItemResponse is returned by the ABS item scan endpoint.
type ScanItemResponse struct {
	Result string `json:"result"`
}

// LibraryItem represents a book, audiobook, podcast, or episode indexed by ABS.
type LibraryItem struct {
	ID                   string        `json:"id"`
	LibraryID            string        `json:"libraryId"`
	FolderID             string        `json:"folderId"`
	Path                 string        `json:"path"`
	RelPath              string        `json:"relPath"`
	IsFile               bool          `json:"isFile"`
	MtimeMs              int64         `json:"mtimeMs"`
	CTimeMs              int64         `json:"ctimeMs"`
	BirthtimeMs          int64         `json:"birthtimeMs"`
	AddedAt              int64         `json:"addedAt"`
	UpdatedAt            int64         `json:"updatedAt"`
	IsMissing            bool          `json:"isMissing"`
	IsInvalid            bool          `json:"isInvalid"`
	MediaType            string        `json:"mediaType"`
	Media                Media         `json:"media"`
	AuthorNamesFirstLast string        `json:"authorNamesFirstLast,omitempty"`
	AuthorNamesLastFirst string        `json:"authorNamesLastFirst,omitempty"`
	LibraryFiles         []LibraryFile `json:"libraryFiles,omitempty"`
}

// Media contains the media metadata embedded in a library item response.
type Media struct {
	ID            string      `json:"id"`
	LibraryItemID string      `json:"libraryItemId"`
	Metadata      Metadata    `json:"metadata"`
	CoverPath     string      `json:"coverPath"`
	EbookFile     *EbookFile  `json:"ebookFile,omitempty"`
	AudioFiles    []AudioFile `json:"audioFiles,omitempty"`
	Tracks        []Track     `json:"tracks,omitempty"`
	Duration      float64     `json:"duration"`
	Size          int64       `json:"size"`
}

// Metadata contains user-facing book or audiobook metadata.
type Metadata struct {
	Title             string   `json:"title"`
	TitleIgnorePrefix string   `json:"titleIgnorePrefix,omitempty"`
	Subtitle          string   `json:"subtitle,omitempty"`
	Authors           []Author `json:"authors,omitempty"`
	AuthorName        string   `json:"authorName,omitempty"`
	AuthorNameLF      string   `json:"authorNameLF,omitempty"`
	Series            []Series `json:"series,omitempty"`
	SeriesName        string   `json:"seriesName,omitempty"`
	SeriesSequence    string   `json:"seriesSequence,omitempty"`
	Description       string   `json:"description,omitempty"`
	Publisher         string   `json:"publisher,omitempty"`
	PublishedYear     string   `json:"publishedYear,omitempty"`
	PublishedDate     string   `json:"publishedDate,omitempty"`
	Language          string   `json:"language,omitempty"`
	Genres            []string `json:"genres,omitempty"`
	Tags              []string `json:"tags,omitempty"`
	ASIN              string   `json:"asin,omitempty"`
	ISBN              string   `json:"isbn,omitempty"`
	Explicit          bool     `json:"explicit"`
	Abridged          bool     `json:"abridged"`
	NarratorName      string   `json:"narratorName,omitempty"`
}

// Author represents an author object in ABS metadata.
type Author struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	ImagePath   string `json:"imagePath,omitempty"`
}

// Series represents a series object in ABS metadata.
type Series struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// LibraryFile represents a file associated with a library item.
type LibraryFile struct {
	Ino       string       `json:"ino"`
	Metadata  FileMetadata `json:"metadata"`
	AddedAt   int64        `json:"addedAt"`
	UpdatedAt int64        `json:"updatedAt"`
	FileType  string       `json:"fileType"`
}

// FileMetadata contains filesystem metadata for a library file.
type FileMetadata struct {
	Filename    string `json:"filename"`
	Path        string `json:"path"`
	RelPath     string `json:"relPath"`
	Size        int64  `json:"size"`
	MtimeMs     int64  `json:"mtimeMs"`
	CtimeMs     int64  `json:"ctimeMs"`
	BirthtimeMs int64  `json:"birthtimeMs"`
}

// AudioFile represents an audio file in an audiobook item.
type AudioFile struct {
	LibraryFile
	TrackNumberFromMeta int    `json:"trackNumFromMeta"`
	DiscNumberFromMeta  int    `json:"discNumFromMeta"`
	Bitrate             int    `json:"bitRate"`
	Codec               string `json:"codec"`
	TimeBase            string `json:"timeBase"`
}

// EbookFile represents an ebook file.
type EbookFile struct {
	LibraryFile
}

// Track represents a track in an audiobook item.
type Track struct {
	Index       int     `json:"index"`
	StartOffset float64 `json:"startOffset"`
	Duration    float64 `json:"duration"`
	Title       string  `json:"title,omitempty"`
	ContentURL  string  `json:"contentUrl"`
}
