import { usePathname, useSearchParams } from "next/navigation";
import { useCallback, useMemo } from "react";
import { VideoOrder, VideoSortBy, VideoType } from "./useVideos";

// Query parameter names used to persist the state of a video list in the URL
const PAGE_PARAM = "page";
const TYPES_PARAM = "types";
const SORT_PARAM = "sort";
const ORDER_PARAM = "order";

const DEFAULT_SORT_BY = VideoSortBy.Date;
const DEFAULT_ORDER = VideoOrder.Desc;

export interface VideoListParams {
  page: number;
  videoTypes: VideoType[];
  sortBy: VideoSortBy;
  order: VideoOrder;
}

const parsePage = (raw: string | null): number => {
  const page = Number(raw);
  return Number.isInteger(page) && page >= 1 ? page : 1;
};

const parseEnumValue = <T extends string>(
  raw: string | null,
  values: readonly T[],
  fallback: T
): T => (raw !== null && values.includes(raw as T) ? (raw as T) : fallback);

/**
 * Keeps the page, video type filter, sort field and sort order of a video list in the
 * URL query string, so they survive navigating to a video and back, reloads, and can be
 * shared as links. Default values are omitted from the URL. Changing a filter resets the
 * page to 1 within the same history entry.
 *
 * The URL is updated with window.history.pushState, which Next.js syncs with
 * useSearchParams. Unlike router.push this causes no server round trip, keeps the scroll
 * position and does not reset document.title to the route metadata.
 */
const useVideoListParams = () => {
  const pathname = usePathname();
  const searchParams = useSearchParams();

  const params = useMemo<VideoListParams>(() => {
    const videoTypeValues = Object.values(VideoType);
    return {
      page: parsePage(searchParams.get(PAGE_PARAM)),
      videoTypes: searchParams
        .getAll(TYPES_PARAM)
        .filter((value): value is VideoType => videoTypeValues.includes(value as VideoType)),
      sortBy: parseEnumValue(searchParams.get(SORT_PARAM), Object.values(VideoSortBy), DEFAULT_SORT_BY),
      order: parseEnumValue(searchParams.get(ORDER_PARAM), Object.values(VideoOrder), DEFAULT_ORDER),
    };
  }, [searchParams]);

  const update = useCallback(
    (changes: Partial<VideoListParams>) => {
      const next = new URLSearchParams(searchParams.toString());

      if (changes.page !== undefined) {
        if (changes.page > 1) next.set(PAGE_PARAM, String(changes.page));
        else next.delete(PAGE_PARAM);
      }
      if (changes.videoTypes !== undefined) {
        next.delete(TYPES_PARAM);
        changes.videoTypes.forEach((type) => next.append(TYPES_PARAM, type));
      }
      if (changes.sortBy !== undefined) {
        if (changes.sortBy !== DEFAULT_SORT_BY) next.set(SORT_PARAM, changes.sortBy);
        else next.delete(SORT_PARAM);
      }
      if (changes.order !== undefined) {
        if (changes.order !== DEFAULT_ORDER) next.set(ORDER_PARAM, changes.order);
        else next.delete(ORDER_PARAM);
      }

      const query = next.toString();
      if (query === searchParams.toString()) return;
      window.history.pushState(null, "", query ? `${pathname}?${query}` : pathname);
    },
    [pathname, searchParams]
  );

  const setPage = useCallback((page: number) => update({ page }), [update]);
  const setVideoTypes = useCallback(
    (videoTypes: VideoType[]) => update({ videoTypes, page: 1 }),
    [update]
  );
  const setSortBy = useCallback((sortBy: VideoSortBy) => update({ sortBy, page: 1 }), [update]);
  const setOrder = useCallback((order: VideoOrder) => update({ order, page: 1 }), [update]);

  return { ...params, setPage, setVideoTypes, setSortBy, setOrder };
};

export { useVideoListParams };
