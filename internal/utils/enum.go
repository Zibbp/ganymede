package utils

type Role string

const (
	AdminRole    Role = "admin"
	EditorRole   Role = "editor"
	ArchiverRole Role = "archiver"
	UserRole     Role = "user"
)

func (Role) Values() (kinds []string) {
	for _, s := range []Role{AdminRole, EditorRole, ArchiverRole, UserRole} {
		kinds = append(kinds, string(s))
	}
	return
}

// IsValidRole checks if a string is a valid Role.
func IsValidRole(role string) bool {
	validRoles := map[string]struct{}{
		string(AdminRole):    {},
		string(EditorRole):   {},
		string(ArchiverRole): {},
		string(UserRole):     {},
	}

	_, exists := validRoles[role]
	return exists
}

// ApiKeyResource is the resource half of an ApiKeyScope. It maps to one of
// the API route groups in internal/transport/http/handler.go.
//
// The catalog is intentionally complete (every existing route group is
// listed) so future migrations of routes to RequireRoleOrScope only need
// to retag the route, not extend this enum. Resources whose routes are
// not yet migrated are flagged "reserved" in the wiki documentation —
// keys can hold those scopes today but no route checks them yet.
type ApiKeyResource string

const (
	// ApiKeyResourceWildcard matches every resource. Pair with a tier to
	// grant cross-resource access (e.g. *:admin = full superuser).
	ApiKeyResourceWildcard ApiKeyResource = "*"

	// Resources whose routes already enforce API key scopes.
	ApiKeyResourceVod      ApiKeyResource = "vod"
	ApiKeyResourcePlaylist ApiKeyResource = "playlist"
	ApiKeyResourceQueue    ApiKeyResource = "queue"

	// Resources for route groups whose write/admin endpoints accept API
	// keys. /chapter, /category and /twitch are intentionally absent —
	// they only expose public reads, so no scope ever gates them.
	// /playback is also absent: it's per-user UX state attributed via
	// the session cookie, not a script-friendly automation surface.
	ApiKeyResourceChannel      ApiKeyResource = "channel"
	ApiKeyResourceArchive      ApiKeyResource = "archive"
	ApiKeyResourceLive         ApiKeyResource = "live"
	ApiKeyResourceUser         ApiKeyResource = "user"
	ApiKeyResourceConfig       ApiKeyResource = "config"
	ApiKeyResourceNotification ApiKeyResource = "notification"
	ApiKeyResourceTask         ApiKeyResource = "task"
	ApiKeyResourceBlockedVideo ApiKeyResource = "blocked_video"

	// ApiKeyResourceSystem covers server-wide stats and info endpoints
	// under /admin/{video-statistics, system-overview, storage-distribution,
	// info}. Named "system" rather than "admin" so the scope string
	// reads cleanly (e.g. system:read instead of admin:admin).
	ApiKeyResourceSystem ApiKeyResource = "system"
)

// AllApiKeyResources lists every defined resource. Used by the validator
// and exposed to the frontend so the create form can offer the catalog.
func AllApiKeyResources() []ApiKeyResource {
	return []ApiKeyResource{
		ApiKeyResourceWildcard,
		ApiKeyResourceVod,
		ApiKeyResourcePlaylist,
		ApiKeyResourceQueue,
		ApiKeyResourceChannel,
		ApiKeyResourceArchive,
		ApiKeyResourceLive,
		ApiKeyResourceUser,
		ApiKeyResourceConfig,
		ApiKeyResourceNotification,
		ApiKeyResourceTask,
		ApiKeyResourceBlockedVideo,
		ApiKeyResourceSystem,
	}
}

// IsValid reports whether r is a defined resource.
func (r ApiKeyResource) IsValid() bool {
	for _, valid := range AllApiKeyResources() {
		if r == valid {
			return true
		}
	}
	return false
}

// ApiKeyTier is the permission level half of an ApiKeyScope. Tiers form a
// hierarchy within a single resource: admin > write > read.
//
// We keep all three tiers (rather than collapsing to read/write) because
// admin gates a meaningful safety boundary — destructive deletes (e.g.
// DELETE /vod/:id) and queue-control actions (POST /queue, /queue/:id/stop).
// A "modify metadata" key needs vod:write; a "wipe old VODs" key needs
// vod:admin.
type ApiKeyTier string

const (
	ApiKeyTierRead  ApiKeyTier = "read"
	ApiKeyTierWrite ApiKeyTier = "write"
	ApiKeyTierAdmin ApiKeyTier = "admin"
)

// AllApiKeyTiers lists every defined tier.
func AllApiKeyTiers() []ApiKeyTier {
	return []ApiKeyTier{ApiKeyTierRead, ApiKeyTierWrite, ApiKeyTierAdmin}
}

// rank gives a tier its position in the read < write < admin hierarchy.
// Returns 0 for any unknown tier so unknown values never satisfy a check.
func (t ApiKeyTier) rank() int {
	switch t {
	case ApiKeyTierRead:
		return 1
	case ApiKeyTierWrite:
		return 2
	case ApiKeyTierAdmin:
		return 3
	}
	return 0
}

// IsValid reports whether t is one of the defined tiers.
func (t ApiKeyTier) IsValid() bool {
	return t.rank() > 0
}

// ApiKeyScope is the on-the-wire string form of a (resource, tier) pair,
// formatted as "<resource>:<tier>" — e.g. "vod:write", "*:admin".
type ApiKeyScope string

// Predeclared scopes. Routes use these in calls to RequireRoleOrScope;
// the create form offers them in its catalog.
const (
	// Wildcard scopes apply across every resource.
	ApiKeyScopeAllRead  ApiKeyScope = "*:read"
	ApiKeyScopeAllWrite ApiKeyScope = "*:write"
	ApiKeyScopeAllAdmin ApiKeyScope = "*:admin"

	// VOD scopes.
	ApiKeyScopeVodRead  ApiKeyScope = "vod:read"
	ApiKeyScopeVodWrite ApiKeyScope = "vod:write"
	ApiKeyScopeVodAdmin ApiKeyScope = "vod:admin"

	// Playlist scopes.
	ApiKeyScopePlaylistRead  ApiKeyScope = "playlist:read"
	ApiKeyScopePlaylistWrite ApiKeyScope = "playlist:write"
	ApiKeyScopePlaylistAdmin ApiKeyScope = "playlist:admin"

	// Queue scopes.
	ApiKeyScopeQueueRead  ApiKeyScope = "queue:read"
	ApiKeyScopeQueueWrite ApiKeyScope = "queue:write"
	ApiKeyScopeQueueAdmin ApiKeyScope = "queue:admin"

	// Per-resource scopes for route groups whose write/admin endpoints
	// accept API keys.
	ApiKeyScopeChannelRead       ApiKeyScope = "channel:read"
	ApiKeyScopeChannelWrite      ApiKeyScope = "channel:write"
	ApiKeyScopeChannelAdmin      ApiKeyScope = "channel:admin"
	ApiKeyScopeArchiveRead       ApiKeyScope = "archive:read"
	ApiKeyScopeArchiveWrite      ApiKeyScope = "archive:write"
	ApiKeyScopeArchiveAdmin      ApiKeyScope = "archive:admin"
	ApiKeyScopeLiveRead          ApiKeyScope = "live:read"
	ApiKeyScopeLiveWrite         ApiKeyScope = "live:write"
	ApiKeyScopeLiveAdmin         ApiKeyScope = "live:admin"
	ApiKeyScopeUserRead          ApiKeyScope = "user:read"
	ApiKeyScopeUserWrite         ApiKeyScope = "user:write"
	ApiKeyScopeUserAdmin         ApiKeyScope = "user:admin"
	ApiKeyScopeConfigRead        ApiKeyScope = "config:read"
	ApiKeyScopeConfigWrite       ApiKeyScope = "config:write"
	ApiKeyScopeConfigAdmin       ApiKeyScope = "config:admin"
	ApiKeyScopeNotificationRead  ApiKeyScope = "notification:read"
	ApiKeyScopeNotificationWrite ApiKeyScope = "notification:write"
	ApiKeyScopeNotificationAdmin ApiKeyScope = "notification:admin"
	ApiKeyScopeTaskRead          ApiKeyScope = "task:read"
	ApiKeyScopeTaskWrite         ApiKeyScope = "task:write"
	ApiKeyScopeTaskAdmin         ApiKeyScope = "task:admin"
	ApiKeyScopeBlockedVideoRead  ApiKeyScope = "blocked_video:read"
	ApiKeyScopeBlockedVideoWrite ApiKeyScope = "blocked_video:write"
	ApiKeyScopeBlockedVideoAdmin ApiKeyScope = "blocked_video:admin"
	ApiKeyScopeSystemRead        ApiKeyScope = "system:read"
	ApiKeyScopeSystemWrite       ApiKeyScope = "system:write"
	ApiKeyScopeSystemAdmin       ApiKeyScope = "system:admin"
)

// MakeApiKeyScope builds a scope from its components.
func MakeApiKeyScope(r ApiKeyResource, t ApiKeyTier) ApiKeyScope {
	return ApiKeyScope(string(r) + ":" + string(t))
}

// Parse splits a scope into its resource and tier. The boolean is false
// if the string is malformed or names an unknown resource/tier.
func (s ApiKeyScope) Parse() (ApiKeyResource, ApiKeyTier, bool) {
	str := string(s)
	colon := -1
	for i := 0; i < len(str); i++ {
		if str[i] == ':' {
			colon = i
			break
		}
	}
	if colon < 1 || colon == len(str)-1 {
		return "", "", false
	}
	r := ApiKeyResource(str[:colon])
	t := ApiKeyTier(str[colon+1:])
	if !r.IsValid() || !t.IsValid() {
		return "", "", false
	}
	return r, t, true
}

// IsValid reports whether s is a well-formed scope naming a defined
// resource and tier.
func (s ApiKeyScope) IsValid() bool {
	_, _, ok := s.Parse()
	return ok
}

// Includes reports whether s grants at least the access named by required.
// A holder scope satisfies a requirement when:
//   - the holder's resource matches the required resource, OR the holder
//     uses the wildcard resource ("*"); AND
//   - the holder's tier rank is greater than or equal to the required tier
//     rank (admin > write > read).
//
// Both s and required must be valid; an unknown resource/tier never
// satisfies a check.
func (s ApiKeyScope) Includes(required ApiKeyScope) bool {
	holderRes, holderTier, ok := s.Parse()
	if !ok {
		return false
	}
	requiredRes, requiredTier, ok := required.Parse()
	if !ok {
		return false
	}
	if holderRes != ApiKeyResourceWildcard && holderRes != requiredRes {
		return false
	}
	return holderTier.rank() >= requiredTier.rank()
}

// AllApiKeyScopes lists every defined scope as a flat slice. Used by the
// service-layer validator to reject unknown scopes on create, and by the
// frontend hooks file to populate the create form's catalog.
func AllApiKeyScopes() []ApiKeyScope {
	out := make([]ApiKeyScope, 0, len(AllApiKeyResources())*len(AllApiKeyTiers()))
	for _, r := range AllApiKeyResources() {
		for _, t := range AllApiKeyTiers() {
			out = append(out, MakeApiKeyScope(r, t))
		}
	}
	return out
}

// IsValidApiKeyScope checks if a string is a defined scope. Convenience
// wrapper for callers that have a string and don't want to convert.
func IsValidApiKeyScope(scope string) bool {
	return ApiKeyScope(scope).IsValid()
}

// ApiKeyScopes is a typed slice with helper methods. A key's permissions
// are the union of its scopes.
type ApiKeyScopes []ApiKeyScope

// Includes reports whether any element of ss satisfies the required scope.
func (ss ApiKeyScopes) Includes(required ApiKeyScope) bool {
	for _, s := range ss {
		if s.Includes(required) {
			return true
		}
	}
	return false
}

// Strings returns the scopes as a []string, useful for ent persistence
// and JSON marshalling.
func (ss ApiKeyScopes) Strings() []string {
	out := make([]string, len(ss))
	for i, s := range ss {
		out[i] = string(s)
	}
	return out
}

// ApiKeyScopesFromStrings converts a []string (typically loaded from ent
// or a request body) into a typed ApiKeyScopes. Invalid scope strings are
// kept as-is so the caller can detect them via IsValid; this lets the
// service layer return precise error messages rather than silently
// dropping entries.
func ApiKeyScopesFromStrings(in []string) ApiKeyScopes {
	out := make(ApiKeyScopes, len(in))
	for i, s := range in {
		out[i] = ApiKeyScope(s)
	}
	return out
}

type VideoPlatform string

const (
	PlatformTwitch  VideoPlatform = "twitch"
	PlatformYoutube VideoPlatform = "youtube"
)

func (VideoPlatform) Values() (kinds []string) {
	for _, s := range []VideoPlatform{PlatformTwitch, PlatformYoutube} {
		kinds = append(kinds, string(s))
	}
	return
}

type VodType string

const (
	Archive   VodType = "archive"
	Live      VodType = "live"
	Highlight VodType = "highlight"
	Upload    VodType = "upload"
	Clip      VodType = "clip"
)

func (VodType) Values() (kinds []string) {
	for _, s := range []VodType{Archive, Live, Highlight, Upload, Clip} {
		kinds = append(kinds, string(s))
	}
	return
}

type VideoSort string

const (
	SortDate       VideoSort = "date"        // streamed at / published at / upload date
	SortViews      VideoSort = "views"       // views from platform
	SortLocalViews VideoSort = "local_views" // views from Ganymede
	SortCreated    VideoSort = "created"     // when the vod was created in Ganymede
)

func (VideoSort) Values() (kinds []string) {
	for _, s := range []VideoSort{SortDate, SortViews, SortLocalViews, SortCreated} {
		kinds = append(kinds, string(s))
	}
	return
}

type SortOrder string

const (
	SortOrderAsc  SortOrder = "asc"
	SortOrderDesc SortOrder = "desc"
)

func (SortOrder) Values() (kinds []string) {
	for _, s := range []SortOrder{SortOrderAsc, SortOrderDesc} {
		kinds = append(kinds, string(s))
	}
	return
}

type TaskStatus string

const (
	Success TaskStatus = "success"
	Running TaskStatus = "running"
	Pending TaskStatus = "pending"
	Failed  TaskStatus = "failed"
)

func (TaskStatus) Values() (kinds []string) {
	for _, s := range []TaskStatus{Success, Running, Pending, Failed} {
		kinds = append(kinds, string(s))
	}
	return
}

type VodQuality string

const (
	Best   VodQuality = "best"
	Source VodQuality = "source"
	R1440  VodQuality = "1440"
	R1080  VodQuality = "1080"
	R720   VodQuality = "720"
	R480   VodQuality = "480"
	R360   VodQuality = "360"
	R160   VodQuality = "160"
	Audio  VodQuality = "audio"
)

func (VodQuality) Values() (kinds []string) {
	for _, s := range []VodQuality{Best, Source, R1440, R1080, R720, R480, R360, R160, Audio} {
		kinds = append(kinds, string(s))
	}
	return
}

func (q VodQuality) String() string {
	return string(q)
}

type PlaybackStatus string

const (
	InProgress PlaybackStatus = "in_progress"
	Finished   PlaybackStatus = "finished"
)

func (PlaybackStatus) Values() (kinds []string) {
	for _, s := range []PlaybackStatus{InProgress, Finished} {
		kinds = append(kinds, string(s))
	}
	return
}

type TaskName string

const (
	TaskCreateFolder             TaskName = "task_vod_create_folder"
	TaskDownloadThumbnail        TaskName = "task_vod_download_thumbnail"
	TaskSaveInfo                 TaskName = "task_vod_save_info"
	TaskDownloadVideo            TaskName = "task_video_download"
	TaskDownloadLiveVideo        TaskName = "task_live_video_download" // not used queue
	TaskPostProcessVideo         TaskName = "task_video_convert"
	TaskMoveVideo                TaskName = "task_video_move"
	TaskDownloadChat             TaskName = "task_chat_download"
	TaskDownloadLiveChat         TaskName = "task_live_chat_download" // not used queue
	TaskConvertChat              TaskName = "task_chat_convert"
	TaskRenderChat               TaskName = "task_chat_render"
	TaskMoveChat                 TaskName = "task_chat_move"
	TaskUpdateLiveStreamMetadata TaskName = "task_update_live_stream_metadata" // not used queue
)

func (TaskName) Values() (kinds []string) {
	for _, s := range []TaskName{TaskCreateFolder, TaskDownloadThumbnail, TaskSaveInfo, TaskDownloadVideo, TaskPostProcessVideo, TaskMoveVideo, TaskDownloadChat, TaskConvertChat, TaskRenderChat, TaskMoveChat, TaskUpdateLiveStreamMetadata} {
		kinds = append(kinds, string(s))
	}
	return
}

func GetTaskName(s string) TaskName {
	switch s {
	case string(TaskCreateFolder):
		return TaskCreateFolder
	case string(TaskDownloadThumbnail):
		return TaskDownloadThumbnail
	case string(TaskSaveInfo):
		return TaskSaveInfo
	case string(TaskDownloadVideo):
		return TaskDownloadVideo
	case string(TaskDownloadLiveVideo):
		return TaskDownloadVideo
	case string(TaskPostProcessVideo):
		return TaskPostProcessVideo
	case string(TaskMoveVideo):
		return TaskMoveVideo
	case string(TaskDownloadChat):
		return TaskDownloadChat
	case string(TaskDownloadLiveChat):
		return TaskDownloadChat
	case string(TaskConvertChat):
		return TaskConvertChat
	case string(TaskRenderChat):
		return TaskRenderChat
	case string(TaskMoveChat):
		return TaskMoveChat
	case string(TaskUpdateLiveStreamMetadata):
		return TaskUpdateLiveStreamMetadata
	default:
		return ""
	}
}

type ProxyType string

const (
	ProxyTypeTwitchHLS ProxyType = "twitch_hls"
	ProxyTypeHTTP      ProxyType = "http"
)

func (ProxyType) Values() (kinds []string) {
	for _, s := range []ProxyType{ProxyTypeTwitchHLS, ProxyTypeHTTP} {
		kinds = append(kinds, string(s))
	}
	return
}

// PlaylistRuleOperator represents the operator used in playlist rules.
// also update http structs when changing this
type PlaylistRuleOperator string

const (
	OperatorEquals   PlaylistRuleOperator = "equals"
	OperatorContains PlaylistRuleOperator = "contains"
	OperatorRegex    PlaylistRuleOperator = "regex"
)

func (PlaylistRuleOperator) Values() (kinds []string) {
	for _, s := range []PlaylistRuleOperator{OperatorEquals, OperatorContains, OperatorRegex} {
		kinds = append(kinds, string(s))
	}
	return
}

// PlaylistField represents the fields that can be used in playlist rules.
// also update http structs when changing this
// need to also run `make ent_generate` to update ent schema
type PlaylistRuleField string

const (
	FieldTitle       PlaylistRuleField = "title"
	FieldCategory    PlaylistRuleField = "category"
	FieldType        PlaylistRuleField = "type"
	FieldPlatform    PlaylistRuleField = "platform"
	FieldChannelName PlaylistRuleField = "channel_name"
)

func (PlaylistRuleField) Values() (kinds []string) {
	for _, s := range []PlaylistRuleField{FieldTitle, FieldCategory, FieldType, FieldPlatform, FieldChannelName} {
		kinds = append(kinds, string(s))
	}
	return
}

// ChapterType represents the type of a chapter.
type ChapterType string

const (
	ChapterTypeGameChange ChapterType = "GAME_CHANGE" // A chapter that indicates a change in the game being played
	ChapterTypeFallback   ChapterType = "FALLBACK"    // A fallback chapter to be used when no other chapter is available, typically the video category/game is used instead
)

func (ChapterType) Values() (kinds []string) {
	for _, s := range []ChapterType{ChapterTypeGameChange, ChapterTypeFallback} {
		kinds = append(kinds, string(s))
	}
	return
}
