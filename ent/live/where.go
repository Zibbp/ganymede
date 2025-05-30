// Code generated by ent, DO NOT EDIT.

package live

import (
	"time"

	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqlgraph"
	"github.com/google/uuid"
	"github.com/zibbp/ganymede/ent/predicate"
)

// ID filters vertices based on their ID field.
func ID(id uuid.UUID) predicate.Live {
	return predicate.Live(sql.FieldEQ(FieldID, id))
}

// IDEQ applies the EQ predicate on the ID field.
func IDEQ(id uuid.UUID) predicate.Live {
	return predicate.Live(sql.FieldEQ(FieldID, id))
}

// IDNEQ applies the NEQ predicate on the ID field.
func IDNEQ(id uuid.UUID) predicate.Live {
	return predicate.Live(sql.FieldNEQ(FieldID, id))
}

// IDIn applies the In predicate on the ID field.
func IDIn(ids ...uuid.UUID) predicate.Live {
	return predicate.Live(sql.FieldIn(FieldID, ids...))
}

// IDNotIn applies the NotIn predicate on the ID field.
func IDNotIn(ids ...uuid.UUID) predicate.Live {
	return predicate.Live(sql.FieldNotIn(FieldID, ids...))
}

// IDGT applies the GT predicate on the ID field.
func IDGT(id uuid.UUID) predicate.Live {
	return predicate.Live(sql.FieldGT(FieldID, id))
}

// IDGTE applies the GTE predicate on the ID field.
func IDGTE(id uuid.UUID) predicate.Live {
	return predicate.Live(sql.FieldGTE(FieldID, id))
}

// IDLT applies the LT predicate on the ID field.
func IDLT(id uuid.UUID) predicate.Live {
	return predicate.Live(sql.FieldLT(FieldID, id))
}

// IDLTE applies the LTE predicate on the ID field.
func IDLTE(id uuid.UUID) predicate.Live {
	return predicate.Live(sql.FieldLTE(FieldID, id))
}

// WatchLive applies equality check predicate on the "watch_live" field. It's identical to WatchLiveEQ.
func WatchLive(v bool) predicate.Live {
	return predicate.Live(sql.FieldEQ(FieldWatchLive, v))
}

// WatchVod applies equality check predicate on the "watch_vod" field. It's identical to WatchVodEQ.
func WatchVod(v bool) predicate.Live {
	return predicate.Live(sql.FieldEQ(FieldWatchVod, v))
}

// DownloadArchives applies equality check predicate on the "download_archives" field. It's identical to DownloadArchivesEQ.
func DownloadArchives(v bool) predicate.Live {
	return predicate.Live(sql.FieldEQ(FieldDownloadArchives, v))
}

// DownloadHighlights applies equality check predicate on the "download_highlights" field. It's identical to DownloadHighlightsEQ.
func DownloadHighlights(v bool) predicate.Live {
	return predicate.Live(sql.FieldEQ(FieldDownloadHighlights, v))
}

// DownloadUploads applies equality check predicate on the "download_uploads" field. It's identical to DownloadUploadsEQ.
func DownloadUploads(v bool) predicate.Live {
	return predicate.Live(sql.FieldEQ(FieldDownloadUploads, v))
}

// DownloadSubOnly applies equality check predicate on the "download_sub_only" field. It's identical to DownloadSubOnlyEQ.
func DownloadSubOnly(v bool) predicate.Live {
	return predicate.Live(sql.FieldEQ(FieldDownloadSubOnly, v))
}

// IsLive applies equality check predicate on the "is_live" field. It's identical to IsLiveEQ.
func IsLive(v bool) predicate.Live {
	return predicate.Live(sql.FieldEQ(FieldIsLive, v))
}

// ArchiveChat applies equality check predicate on the "archive_chat" field. It's identical to ArchiveChatEQ.
func ArchiveChat(v bool) predicate.Live {
	return predicate.Live(sql.FieldEQ(FieldArchiveChat, v))
}

// Resolution applies equality check predicate on the "resolution" field. It's identical to ResolutionEQ.
func Resolution(v string) predicate.Live {
	return predicate.Live(sql.FieldEQ(FieldResolution, v))
}

// LastLive applies equality check predicate on the "last_live" field. It's identical to LastLiveEQ.
func LastLive(v time.Time) predicate.Live {
	return predicate.Live(sql.FieldEQ(FieldLastLive, v))
}

// RenderChat applies equality check predicate on the "render_chat" field. It's identical to RenderChatEQ.
func RenderChat(v bool) predicate.Live {
	return predicate.Live(sql.FieldEQ(FieldRenderChat, v))
}

// VideoAge applies equality check predicate on the "video_age" field. It's identical to VideoAgeEQ.
func VideoAge(v int64) predicate.Live {
	return predicate.Live(sql.FieldEQ(FieldVideoAge, v))
}

// ApplyCategoriesToLive applies equality check predicate on the "apply_categories_to_live" field. It's identical to ApplyCategoriesToLiveEQ.
func ApplyCategoriesToLive(v bool) predicate.Live {
	return predicate.Live(sql.FieldEQ(FieldApplyCategoriesToLive, v))
}

// WatchClips applies equality check predicate on the "watch_clips" field. It's identical to WatchClipsEQ.
func WatchClips(v bool) predicate.Live {
	return predicate.Live(sql.FieldEQ(FieldWatchClips, v))
}

// ClipsLimit applies equality check predicate on the "clips_limit" field. It's identical to ClipsLimitEQ.
func ClipsLimit(v int) predicate.Live {
	return predicate.Live(sql.FieldEQ(FieldClipsLimit, v))
}

// ClipsIntervalDays applies equality check predicate on the "clips_interval_days" field. It's identical to ClipsIntervalDaysEQ.
func ClipsIntervalDays(v int) predicate.Live {
	return predicate.Live(sql.FieldEQ(FieldClipsIntervalDays, v))
}

// ClipsLastChecked applies equality check predicate on the "clips_last_checked" field. It's identical to ClipsLastCheckedEQ.
func ClipsLastChecked(v time.Time) predicate.Live {
	return predicate.Live(sql.FieldEQ(FieldClipsLastChecked, v))
}

// ClipsIgnoreLastChecked applies equality check predicate on the "clips_ignore_last_checked" field. It's identical to ClipsIgnoreLastCheckedEQ.
func ClipsIgnoreLastChecked(v bool) predicate.Live {
	return predicate.Live(sql.FieldEQ(FieldClipsIgnoreLastChecked, v))
}

// UpdateMetadataMinutes applies equality check predicate on the "update_metadata_minutes" field. It's identical to UpdateMetadataMinutesEQ.
func UpdateMetadataMinutes(v int) predicate.Live {
	return predicate.Live(sql.FieldEQ(FieldUpdateMetadataMinutes, v))
}

// UpdatedAt applies equality check predicate on the "updated_at" field. It's identical to UpdatedAtEQ.
func UpdatedAt(v time.Time) predicate.Live {
	return predicate.Live(sql.FieldEQ(FieldUpdatedAt, v))
}

// CreatedAt applies equality check predicate on the "created_at" field. It's identical to CreatedAtEQ.
func CreatedAt(v time.Time) predicate.Live {
	return predicate.Live(sql.FieldEQ(FieldCreatedAt, v))
}

// WatchLiveEQ applies the EQ predicate on the "watch_live" field.
func WatchLiveEQ(v bool) predicate.Live {
	return predicate.Live(sql.FieldEQ(FieldWatchLive, v))
}

// WatchLiveNEQ applies the NEQ predicate on the "watch_live" field.
func WatchLiveNEQ(v bool) predicate.Live {
	return predicate.Live(sql.FieldNEQ(FieldWatchLive, v))
}

// WatchVodEQ applies the EQ predicate on the "watch_vod" field.
func WatchVodEQ(v bool) predicate.Live {
	return predicate.Live(sql.FieldEQ(FieldWatchVod, v))
}

// WatchVodNEQ applies the NEQ predicate on the "watch_vod" field.
func WatchVodNEQ(v bool) predicate.Live {
	return predicate.Live(sql.FieldNEQ(FieldWatchVod, v))
}

// DownloadArchivesEQ applies the EQ predicate on the "download_archives" field.
func DownloadArchivesEQ(v bool) predicate.Live {
	return predicate.Live(sql.FieldEQ(FieldDownloadArchives, v))
}

// DownloadArchivesNEQ applies the NEQ predicate on the "download_archives" field.
func DownloadArchivesNEQ(v bool) predicate.Live {
	return predicate.Live(sql.FieldNEQ(FieldDownloadArchives, v))
}

// DownloadHighlightsEQ applies the EQ predicate on the "download_highlights" field.
func DownloadHighlightsEQ(v bool) predicate.Live {
	return predicate.Live(sql.FieldEQ(FieldDownloadHighlights, v))
}

// DownloadHighlightsNEQ applies the NEQ predicate on the "download_highlights" field.
func DownloadHighlightsNEQ(v bool) predicate.Live {
	return predicate.Live(sql.FieldNEQ(FieldDownloadHighlights, v))
}

// DownloadUploadsEQ applies the EQ predicate on the "download_uploads" field.
func DownloadUploadsEQ(v bool) predicate.Live {
	return predicate.Live(sql.FieldEQ(FieldDownloadUploads, v))
}

// DownloadUploadsNEQ applies the NEQ predicate on the "download_uploads" field.
func DownloadUploadsNEQ(v bool) predicate.Live {
	return predicate.Live(sql.FieldNEQ(FieldDownloadUploads, v))
}

// DownloadSubOnlyEQ applies the EQ predicate on the "download_sub_only" field.
func DownloadSubOnlyEQ(v bool) predicate.Live {
	return predicate.Live(sql.FieldEQ(FieldDownloadSubOnly, v))
}

// DownloadSubOnlyNEQ applies the NEQ predicate on the "download_sub_only" field.
func DownloadSubOnlyNEQ(v bool) predicate.Live {
	return predicate.Live(sql.FieldNEQ(FieldDownloadSubOnly, v))
}

// IsLiveEQ applies the EQ predicate on the "is_live" field.
func IsLiveEQ(v bool) predicate.Live {
	return predicate.Live(sql.FieldEQ(FieldIsLive, v))
}

// IsLiveNEQ applies the NEQ predicate on the "is_live" field.
func IsLiveNEQ(v bool) predicate.Live {
	return predicate.Live(sql.FieldNEQ(FieldIsLive, v))
}

// ArchiveChatEQ applies the EQ predicate on the "archive_chat" field.
func ArchiveChatEQ(v bool) predicate.Live {
	return predicate.Live(sql.FieldEQ(FieldArchiveChat, v))
}

// ArchiveChatNEQ applies the NEQ predicate on the "archive_chat" field.
func ArchiveChatNEQ(v bool) predicate.Live {
	return predicate.Live(sql.FieldNEQ(FieldArchiveChat, v))
}

// ResolutionEQ applies the EQ predicate on the "resolution" field.
func ResolutionEQ(v string) predicate.Live {
	return predicate.Live(sql.FieldEQ(FieldResolution, v))
}

// ResolutionNEQ applies the NEQ predicate on the "resolution" field.
func ResolutionNEQ(v string) predicate.Live {
	return predicate.Live(sql.FieldNEQ(FieldResolution, v))
}

// ResolutionIn applies the In predicate on the "resolution" field.
func ResolutionIn(vs ...string) predicate.Live {
	return predicate.Live(sql.FieldIn(FieldResolution, vs...))
}

// ResolutionNotIn applies the NotIn predicate on the "resolution" field.
func ResolutionNotIn(vs ...string) predicate.Live {
	return predicate.Live(sql.FieldNotIn(FieldResolution, vs...))
}

// ResolutionGT applies the GT predicate on the "resolution" field.
func ResolutionGT(v string) predicate.Live {
	return predicate.Live(sql.FieldGT(FieldResolution, v))
}

// ResolutionGTE applies the GTE predicate on the "resolution" field.
func ResolutionGTE(v string) predicate.Live {
	return predicate.Live(sql.FieldGTE(FieldResolution, v))
}

// ResolutionLT applies the LT predicate on the "resolution" field.
func ResolutionLT(v string) predicate.Live {
	return predicate.Live(sql.FieldLT(FieldResolution, v))
}

// ResolutionLTE applies the LTE predicate on the "resolution" field.
func ResolutionLTE(v string) predicate.Live {
	return predicate.Live(sql.FieldLTE(FieldResolution, v))
}

// ResolutionContains applies the Contains predicate on the "resolution" field.
func ResolutionContains(v string) predicate.Live {
	return predicate.Live(sql.FieldContains(FieldResolution, v))
}

// ResolutionHasPrefix applies the HasPrefix predicate on the "resolution" field.
func ResolutionHasPrefix(v string) predicate.Live {
	return predicate.Live(sql.FieldHasPrefix(FieldResolution, v))
}

// ResolutionHasSuffix applies the HasSuffix predicate on the "resolution" field.
func ResolutionHasSuffix(v string) predicate.Live {
	return predicate.Live(sql.FieldHasSuffix(FieldResolution, v))
}

// ResolutionIsNil applies the IsNil predicate on the "resolution" field.
func ResolutionIsNil() predicate.Live {
	return predicate.Live(sql.FieldIsNull(FieldResolution))
}

// ResolutionNotNil applies the NotNil predicate on the "resolution" field.
func ResolutionNotNil() predicate.Live {
	return predicate.Live(sql.FieldNotNull(FieldResolution))
}

// ResolutionEqualFold applies the EqualFold predicate on the "resolution" field.
func ResolutionEqualFold(v string) predicate.Live {
	return predicate.Live(sql.FieldEqualFold(FieldResolution, v))
}

// ResolutionContainsFold applies the ContainsFold predicate on the "resolution" field.
func ResolutionContainsFold(v string) predicate.Live {
	return predicate.Live(sql.FieldContainsFold(FieldResolution, v))
}

// LastLiveEQ applies the EQ predicate on the "last_live" field.
func LastLiveEQ(v time.Time) predicate.Live {
	return predicate.Live(sql.FieldEQ(FieldLastLive, v))
}

// LastLiveNEQ applies the NEQ predicate on the "last_live" field.
func LastLiveNEQ(v time.Time) predicate.Live {
	return predicate.Live(sql.FieldNEQ(FieldLastLive, v))
}

// LastLiveIn applies the In predicate on the "last_live" field.
func LastLiveIn(vs ...time.Time) predicate.Live {
	return predicate.Live(sql.FieldIn(FieldLastLive, vs...))
}

// LastLiveNotIn applies the NotIn predicate on the "last_live" field.
func LastLiveNotIn(vs ...time.Time) predicate.Live {
	return predicate.Live(sql.FieldNotIn(FieldLastLive, vs...))
}

// LastLiveGT applies the GT predicate on the "last_live" field.
func LastLiveGT(v time.Time) predicate.Live {
	return predicate.Live(sql.FieldGT(FieldLastLive, v))
}

// LastLiveGTE applies the GTE predicate on the "last_live" field.
func LastLiveGTE(v time.Time) predicate.Live {
	return predicate.Live(sql.FieldGTE(FieldLastLive, v))
}

// LastLiveLT applies the LT predicate on the "last_live" field.
func LastLiveLT(v time.Time) predicate.Live {
	return predicate.Live(sql.FieldLT(FieldLastLive, v))
}

// LastLiveLTE applies the LTE predicate on the "last_live" field.
func LastLiveLTE(v time.Time) predicate.Live {
	return predicate.Live(sql.FieldLTE(FieldLastLive, v))
}

// RenderChatEQ applies the EQ predicate on the "render_chat" field.
func RenderChatEQ(v bool) predicate.Live {
	return predicate.Live(sql.FieldEQ(FieldRenderChat, v))
}

// RenderChatNEQ applies the NEQ predicate on the "render_chat" field.
func RenderChatNEQ(v bool) predicate.Live {
	return predicate.Live(sql.FieldNEQ(FieldRenderChat, v))
}

// VideoAgeEQ applies the EQ predicate on the "video_age" field.
func VideoAgeEQ(v int64) predicate.Live {
	return predicate.Live(sql.FieldEQ(FieldVideoAge, v))
}

// VideoAgeNEQ applies the NEQ predicate on the "video_age" field.
func VideoAgeNEQ(v int64) predicate.Live {
	return predicate.Live(sql.FieldNEQ(FieldVideoAge, v))
}

// VideoAgeIn applies the In predicate on the "video_age" field.
func VideoAgeIn(vs ...int64) predicate.Live {
	return predicate.Live(sql.FieldIn(FieldVideoAge, vs...))
}

// VideoAgeNotIn applies the NotIn predicate on the "video_age" field.
func VideoAgeNotIn(vs ...int64) predicate.Live {
	return predicate.Live(sql.FieldNotIn(FieldVideoAge, vs...))
}

// VideoAgeGT applies the GT predicate on the "video_age" field.
func VideoAgeGT(v int64) predicate.Live {
	return predicate.Live(sql.FieldGT(FieldVideoAge, v))
}

// VideoAgeGTE applies the GTE predicate on the "video_age" field.
func VideoAgeGTE(v int64) predicate.Live {
	return predicate.Live(sql.FieldGTE(FieldVideoAge, v))
}

// VideoAgeLT applies the LT predicate on the "video_age" field.
func VideoAgeLT(v int64) predicate.Live {
	return predicate.Live(sql.FieldLT(FieldVideoAge, v))
}

// VideoAgeLTE applies the LTE predicate on the "video_age" field.
func VideoAgeLTE(v int64) predicate.Live {
	return predicate.Live(sql.FieldLTE(FieldVideoAge, v))
}

// ApplyCategoriesToLiveEQ applies the EQ predicate on the "apply_categories_to_live" field.
func ApplyCategoriesToLiveEQ(v bool) predicate.Live {
	return predicate.Live(sql.FieldEQ(FieldApplyCategoriesToLive, v))
}

// ApplyCategoriesToLiveNEQ applies the NEQ predicate on the "apply_categories_to_live" field.
func ApplyCategoriesToLiveNEQ(v bool) predicate.Live {
	return predicate.Live(sql.FieldNEQ(FieldApplyCategoriesToLive, v))
}

// WatchClipsEQ applies the EQ predicate on the "watch_clips" field.
func WatchClipsEQ(v bool) predicate.Live {
	return predicate.Live(sql.FieldEQ(FieldWatchClips, v))
}

// WatchClipsNEQ applies the NEQ predicate on the "watch_clips" field.
func WatchClipsNEQ(v bool) predicate.Live {
	return predicate.Live(sql.FieldNEQ(FieldWatchClips, v))
}

// ClipsLimitEQ applies the EQ predicate on the "clips_limit" field.
func ClipsLimitEQ(v int) predicate.Live {
	return predicate.Live(sql.FieldEQ(FieldClipsLimit, v))
}

// ClipsLimitNEQ applies the NEQ predicate on the "clips_limit" field.
func ClipsLimitNEQ(v int) predicate.Live {
	return predicate.Live(sql.FieldNEQ(FieldClipsLimit, v))
}

// ClipsLimitIn applies the In predicate on the "clips_limit" field.
func ClipsLimitIn(vs ...int) predicate.Live {
	return predicate.Live(sql.FieldIn(FieldClipsLimit, vs...))
}

// ClipsLimitNotIn applies the NotIn predicate on the "clips_limit" field.
func ClipsLimitNotIn(vs ...int) predicate.Live {
	return predicate.Live(sql.FieldNotIn(FieldClipsLimit, vs...))
}

// ClipsLimitGT applies the GT predicate on the "clips_limit" field.
func ClipsLimitGT(v int) predicate.Live {
	return predicate.Live(sql.FieldGT(FieldClipsLimit, v))
}

// ClipsLimitGTE applies the GTE predicate on the "clips_limit" field.
func ClipsLimitGTE(v int) predicate.Live {
	return predicate.Live(sql.FieldGTE(FieldClipsLimit, v))
}

// ClipsLimitLT applies the LT predicate on the "clips_limit" field.
func ClipsLimitLT(v int) predicate.Live {
	return predicate.Live(sql.FieldLT(FieldClipsLimit, v))
}

// ClipsLimitLTE applies the LTE predicate on the "clips_limit" field.
func ClipsLimitLTE(v int) predicate.Live {
	return predicate.Live(sql.FieldLTE(FieldClipsLimit, v))
}

// ClipsIntervalDaysEQ applies the EQ predicate on the "clips_interval_days" field.
func ClipsIntervalDaysEQ(v int) predicate.Live {
	return predicate.Live(sql.FieldEQ(FieldClipsIntervalDays, v))
}

// ClipsIntervalDaysNEQ applies the NEQ predicate on the "clips_interval_days" field.
func ClipsIntervalDaysNEQ(v int) predicate.Live {
	return predicate.Live(sql.FieldNEQ(FieldClipsIntervalDays, v))
}

// ClipsIntervalDaysIn applies the In predicate on the "clips_interval_days" field.
func ClipsIntervalDaysIn(vs ...int) predicate.Live {
	return predicate.Live(sql.FieldIn(FieldClipsIntervalDays, vs...))
}

// ClipsIntervalDaysNotIn applies the NotIn predicate on the "clips_interval_days" field.
func ClipsIntervalDaysNotIn(vs ...int) predicate.Live {
	return predicate.Live(sql.FieldNotIn(FieldClipsIntervalDays, vs...))
}

// ClipsIntervalDaysGT applies the GT predicate on the "clips_interval_days" field.
func ClipsIntervalDaysGT(v int) predicate.Live {
	return predicate.Live(sql.FieldGT(FieldClipsIntervalDays, v))
}

// ClipsIntervalDaysGTE applies the GTE predicate on the "clips_interval_days" field.
func ClipsIntervalDaysGTE(v int) predicate.Live {
	return predicate.Live(sql.FieldGTE(FieldClipsIntervalDays, v))
}

// ClipsIntervalDaysLT applies the LT predicate on the "clips_interval_days" field.
func ClipsIntervalDaysLT(v int) predicate.Live {
	return predicate.Live(sql.FieldLT(FieldClipsIntervalDays, v))
}

// ClipsIntervalDaysLTE applies the LTE predicate on the "clips_interval_days" field.
func ClipsIntervalDaysLTE(v int) predicate.Live {
	return predicate.Live(sql.FieldLTE(FieldClipsIntervalDays, v))
}

// ClipsLastCheckedEQ applies the EQ predicate on the "clips_last_checked" field.
func ClipsLastCheckedEQ(v time.Time) predicate.Live {
	return predicate.Live(sql.FieldEQ(FieldClipsLastChecked, v))
}

// ClipsLastCheckedNEQ applies the NEQ predicate on the "clips_last_checked" field.
func ClipsLastCheckedNEQ(v time.Time) predicate.Live {
	return predicate.Live(sql.FieldNEQ(FieldClipsLastChecked, v))
}

// ClipsLastCheckedIn applies the In predicate on the "clips_last_checked" field.
func ClipsLastCheckedIn(vs ...time.Time) predicate.Live {
	return predicate.Live(sql.FieldIn(FieldClipsLastChecked, vs...))
}

// ClipsLastCheckedNotIn applies the NotIn predicate on the "clips_last_checked" field.
func ClipsLastCheckedNotIn(vs ...time.Time) predicate.Live {
	return predicate.Live(sql.FieldNotIn(FieldClipsLastChecked, vs...))
}

// ClipsLastCheckedGT applies the GT predicate on the "clips_last_checked" field.
func ClipsLastCheckedGT(v time.Time) predicate.Live {
	return predicate.Live(sql.FieldGT(FieldClipsLastChecked, v))
}

// ClipsLastCheckedGTE applies the GTE predicate on the "clips_last_checked" field.
func ClipsLastCheckedGTE(v time.Time) predicate.Live {
	return predicate.Live(sql.FieldGTE(FieldClipsLastChecked, v))
}

// ClipsLastCheckedLT applies the LT predicate on the "clips_last_checked" field.
func ClipsLastCheckedLT(v time.Time) predicate.Live {
	return predicate.Live(sql.FieldLT(FieldClipsLastChecked, v))
}

// ClipsLastCheckedLTE applies the LTE predicate on the "clips_last_checked" field.
func ClipsLastCheckedLTE(v time.Time) predicate.Live {
	return predicate.Live(sql.FieldLTE(FieldClipsLastChecked, v))
}

// ClipsLastCheckedIsNil applies the IsNil predicate on the "clips_last_checked" field.
func ClipsLastCheckedIsNil() predicate.Live {
	return predicate.Live(sql.FieldIsNull(FieldClipsLastChecked))
}

// ClipsLastCheckedNotNil applies the NotNil predicate on the "clips_last_checked" field.
func ClipsLastCheckedNotNil() predicate.Live {
	return predicate.Live(sql.FieldNotNull(FieldClipsLastChecked))
}

// ClipsIgnoreLastCheckedEQ applies the EQ predicate on the "clips_ignore_last_checked" field.
func ClipsIgnoreLastCheckedEQ(v bool) predicate.Live {
	return predicate.Live(sql.FieldEQ(FieldClipsIgnoreLastChecked, v))
}

// ClipsIgnoreLastCheckedNEQ applies the NEQ predicate on the "clips_ignore_last_checked" field.
func ClipsIgnoreLastCheckedNEQ(v bool) predicate.Live {
	return predicate.Live(sql.FieldNEQ(FieldClipsIgnoreLastChecked, v))
}

// UpdateMetadataMinutesEQ applies the EQ predicate on the "update_metadata_minutes" field.
func UpdateMetadataMinutesEQ(v int) predicate.Live {
	return predicate.Live(sql.FieldEQ(FieldUpdateMetadataMinutes, v))
}

// UpdateMetadataMinutesNEQ applies the NEQ predicate on the "update_metadata_minutes" field.
func UpdateMetadataMinutesNEQ(v int) predicate.Live {
	return predicate.Live(sql.FieldNEQ(FieldUpdateMetadataMinutes, v))
}

// UpdateMetadataMinutesIn applies the In predicate on the "update_metadata_minutes" field.
func UpdateMetadataMinutesIn(vs ...int) predicate.Live {
	return predicate.Live(sql.FieldIn(FieldUpdateMetadataMinutes, vs...))
}

// UpdateMetadataMinutesNotIn applies the NotIn predicate on the "update_metadata_minutes" field.
func UpdateMetadataMinutesNotIn(vs ...int) predicate.Live {
	return predicate.Live(sql.FieldNotIn(FieldUpdateMetadataMinutes, vs...))
}

// UpdateMetadataMinutesGT applies the GT predicate on the "update_metadata_minutes" field.
func UpdateMetadataMinutesGT(v int) predicate.Live {
	return predicate.Live(sql.FieldGT(FieldUpdateMetadataMinutes, v))
}

// UpdateMetadataMinutesGTE applies the GTE predicate on the "update_metadata_minutes" field.
func UpdateMetadataMinutesGTE(v int) predicate.Live {
	return predicate.Live(sql.FieldGTE(FieldUpdateMetadataMinutes, v))
}

// UpdateMetadataMinutesLT applies the LT predicate on the "update_metadata_minutes" field.
func UpdateMetadataMinutesLT(v int) predicate.Live {
	return predicate.Live(sql.FieldLT(FieldUpdateMetadataMinutes, v))
}

// UpdateMetadataMinutesLTE applies the LTE predicate on the "update_metadata_minutes" field.
func UpdateMetadataMinutesLTE(v int) predicate.Live {
	return predicate.Live(sql.FieldLTE(FieldUpdateMetadataMinutes, v))
}

// UpdatedAtEQ applies the EQ predicate on the "updated_at" field.
func UpdatedAtEQ(v time.Time) predicate.Live {
	return predicate.Live(sql.FieldEQ(FieldUpdatedAt, v))
}

// UpdatedAtNEQ applies the NEQ predicate on the "updated_at" field.
func UpdatedAtNEQ(v time.Time) predicate.Live {
	return predicate.Live(sql.FieldNEQ(FieldUpdatedAt, v))
}

// UpdatedAtIn applies the In predicate on the "updated_at" field.
func UpdatedAtIn(vs ...time.Time) predicate.Live {
	return predicate.Live(sql.FieldIn(FieldUpdatedAt, vs...))
}

// UpdatedAtNotIn applies the NotIn predicate on the "updated_at" field.
func UpdatedAtNotIn(vs ...time.Time) predicate.Live {
	return predicate.Live(sql.FieldNotIn(FieldUpdatedAt, vs...))
}

// UpdatedAtGT applies the GT predicate on the "updated_at" field.
func UpdatedAtGT(v time.Time) predicate.Live {
	return predicate.Live(sql.FieldGT(FieldUpdatedAt, v))
}

// UpdatedAtGTE applies the GTE predicate on the "updated_at" field.
func UpdatedAtGTE(v time.Time) predicate.Live {
	return predicate.Live(sql.FieldGTE(FieldUpdatedAt, v))
}

// UpdatedAtLT applies the LT predicate on the "updated_at" field.
func UpdatedAtLT(v time.Time) predicate.Live {
	return predicate.Live(sql.FieldLT(FieldUpdatedAt, v))
}

// UpdatedAtLTE applies the LTE predicate on the "updated_at" field.
func UpdatedAtLTE(v time.Time) predicate.Live {
	return predicate.Live(sql.FieldLTE(FieldUpdatedAt, v))
}

// CreatedAtEQ applies the EQ predicate on the "created_at" field.
func CreatedAtEQ(v time.Time) predicate.Live {
	return predicate.Live(sql.FieldEQ(FieldCreatedAt, v))
}

// CreatedAtNEQ applies the NEQ predicate on the "created_at" field.
func CreatedAtNEQ(v time.Time) predicate.Live {
	return predicate.Live(sql.FieldNEQ(FieldCreatedAt, v))
}

// CreatedAtIn applies the In predicate on the "created_at" field.
func CreatedAtIn(vs ...time.Time) predicate.Live {
	return predicate.Live(sql.FieldIn(FieldCreatedAt, vs...))
}

// CreatedAtNotIn applies the NotIn predicate on the "created_at" field.
func CreatedAtNotIn(vs ...time.Time) predicate.Live {
	return predicate.Live(sql.FieldNotIn(FieldCreatedAt, vs...))
}

// CreatedAtGT applies the GT predicate on the "created_at" field.
func CreatedAtGT(v time.Time) predicate.Live {
	return predicate.Live(sql.FieldGT(FieldCreatedAt, v))
}

// CreatedAtGTE applies the GTE predicate on the "created_at" field.
func CreatedAtGTE(v time.Time) predicate.Live {
	return predicate.Live(sql.FieldGTE(FieldCreatedAt, v))
}

// CreatedAtLT applies the LT predicate on the "created_at" field.
func CreatedAtLT(v time.Time) predicate.Live {
	return predicate.Live(sql.FieldLT(FieldCreatedAt, v))
}

// CreatedAtLTE applies the LTE predicate on the "created_at" field.
func CreatedAtLTE(v time.Time) predicate.Live {
	return predicate.Live(sql.FieldLTE(FieldCreatedAt, v))
}

// HasChannel applies the HasEdge predicate on the "channel" edge.
func HasChannel() predicate.Live {
	return predicate.Live(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.Edge(sqlgraph.M2O, true, ChannelTable, ChannelColumn),
		)
		sqlgraph.HasNeighbors(s, step)
	})
}

// HasChannelWith applies the HasEdge predicate on the "channel" edge with a given conditions (other predicates).
func HasChannelWith(preds ...predicate.Channel) predicate.Live {
	return predicate.Live(func(s *sql.Selector) {
		step := newChannelStep()
		sqlgraph.HasNeighborsWith(s, step, func(s *sql.Selector) {
			for _, p := range preds {
				p(s)
			}
		})
	})
}

// HasCategories applies the HasEdge predicate on the "categories" edge.
func HasCategories() predicate.Live {
	return predicate.Live(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.Edge(sqlgraph.O2M, false, CategoriesTable, CategoriesColumn),
		)
		sqlgraph.HasNeighbors(s, step)
	})
}

// HasCategoriesWith applies the HasEdge predicate on the "categories" edge with a given conditions (other predicates).
func HasCategoriesWith(preds ...predicate.LiveCategory) predicate.Live {
	return predicate.Live(func(s *sql.Selector) {
		step := newCategoriesStep()
		sqlgraph.HasNeighborsWith(s, step, func(s *sql.Selector) {
			for _, p := range preds {
				p(s)
			}
		})
	})
}

// HasTitleRegex applies the HasEdge predicate on the "title_regex" edge.
func HasTitleRegex() predicate.Live {
	return predicate.Live(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.Edge(sqlgraph.O2M, false, TitleRegexTable, TitleRegexColumn),
		)
		sqlgraph.HasNeighbors(s, step)
	})
}

// HasTitleRegexWith applies the HasEdge predicate on the "title_regex" edge with a given conditions (other predicates).
func HasTitleRegexWith(preds ...predicate.LiveTitleRegex) predicate.Live {
	return predicate.Live(func(s *sql.Selector) {
		step := newTitleRegexStep()
		sqlgraph.HasNeighborsWith(s, step, func(s *sql.Selector) {
			for _, p := range preds {
				p(s)
			}
		})
	})
}

// And groups predicates with the AND operator between them.
func And(predicates ...predicate.Live) predicate.Live {
	return predicate.Live(sql.AndPredicates(predicates...))
}

// Or groups predicates with the OR operator between them.
func Or(predicates ...predicate.Live) predicate.Live {
	return predicate.Live(sql.OrPredicates(predicates...))
}

// Not applies the not operator on the given predicate.
func Not(p predicate.Live) predicate.Live {
	return predicate.Live(sql.NotPredicates(p))
}
