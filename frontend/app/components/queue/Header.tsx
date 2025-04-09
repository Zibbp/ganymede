import { Queue } from "@/app/hooks/useQueue";
import { escapeURL } from "@/app/util/util";
import { env } from "next-runtime-env";
import { Container, Divider, Flex, Image, Text, Tooltip } from "@mantine/core";
import dayjs from "dayjs";
import classes from "./Header.module.css"
import { useTranslations } from "next-intl";

interface Params {
  queue: Queue;
}

const QueueHeader = ({ queue }: Params) => {
  const t = useTranslations('QueueComponents')
  return (
    <div className={classes.queueHeader}>
      <Container size="4xl">
        <div className={classes.queueHeaderContents}>
          <div>
            <Image
              src={
                (env('NEXT_PUBLIC_CDN_URL') ?? '') +
                escapeURL(queue.edges.vod.web_thumbnail_path)
              }
              w={160}
              alt={queue.edges.vod.title}
            />
          </div>
          <div className={classes.queueHeaderRight}>
            <div>
              <Tooltip label={queue.edges.vod.title}>
                <Text lineClamp={1} className={classes.queueHeaderTitle}>
                  {queue.edges.vod.title}
                </Text>
              </Tooltip>
            </div>
            <Flex>
              <Tooltip label={t('externalPlatformVideoIdLabel')}>
                <Text
                  className={classes.queueHeaderHoverText}
                >
                  {queue.edges.vod.ext_id}
                </Text>
              </Tooltip>
              <Divider mx={5} orientation="vertical" />
              <Tooltip label={t('ganymedeVideoIdLabel')}>
                <Text
                  className={classes.queueHeaderHoverText}
                >
                  {queue.edges.vod.id}
                </Text>
              </Tooltip>
            </Flex>
            <Flex>
              {queue.live_archive && (
                <Text className={classes.liveArchive}>Live Archive</Text>
              )}

              {queue.on_hold && <Text className={classes.onHold}>On Hold</Text>}

              <Tooltip label={t('streamedAtLabel')}>
                <Text
                  className={classes.queueHeaderHoverText}
                >
                  {dayjs(queue.edges.vod.streamed_at).format("YYYY/MM/DD")}
                </Text>
              </Tooltip>

            </Flex>
          </div>
        </div>
      </Container>
    </div>
  );
}

export default QueueHeader;