"use client"
import QueueGeneralTimeline from "@/app/components/queue/GeneralTimeline";
import QueueHeader from "@/app/components/queue/Header";
import QueueVideoTimeline from "@/app/components/queue/VideoTimeline";
import GanymedeLoadingText from "@/app/components/utils/GanymedeLoadingText";
import { useAxiosPrivate } from "@/app/hooks/useAxios";
import { useGetQueueItem } from "@/app/hooks/useQueue";
import { Center } from "@mantine/core";
import React, { useEffect } from "react";
import classes from "./QueueIdPage.module.css"
import QueueChatTimeline from "@/app/components/queue/ChatTimeline";
import { useTranslations } from "next-intl";

interface Params {
  id: string;
}

const QueueIdPage = ({ params }: { params: Promise<Params> }) => {
  const { id } = React.use(params);
  const t = useTranslations("QueuePage");
  useEffect(() => {
    document.title = `${t('title')} - ${id}`;
  }, [id]);

  const axiosPrivate = useAxiosPrivate()

  const { data, isPending, isError } = useGetQueueItem(axiosPrivate, id, {
    refetchInterval: 1000
  })

  if (isPending) return (
    <GanymedeLoadingText message={t('loading')} />
  )
  if (isError) return <div>{t('error')}</div>

  return (
    <div>
      <QueueHeader queue={data} />
      <div>
        <Center pt={25}>
          <QueueGeneralTimeline queue={data} />
        </Center>
      </div>
      <Center>
        <div className={classes.timelineBottom}>
          <div
            style={{ paddingTop: "25px" }}
            className={classes.videoTimeline}
          >
            <QueueVideoTimeline queue={data} />
          </div>
          <div style={{ paddingTop: "25px" }}>
            <QueueChatTimeline queue={data} />
          </div>
        </div>
      </Center>
    </div>
  );
}

export default QueueIdPage;