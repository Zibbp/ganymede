"use client"
import GanymedeLoadingText from "@/app/components/utils/GanymedeLoadingText";
import { useAxiosPrivate } from "@/app/hooks/useAxios";
import { QueueLogType, useGetQueueLogs } from "@/app/hooks/useQueue";
import { Box } from "@mantine/core";
import { useSearchParams } from "next/navigation";
import React, { useEffect, useRef } from "react";
import classes from "./QueueLogsPage.module.css"
import { useTranslations } from "next-intl";

interface Params {
  id: string;
}

const QueueLogsPage = ({ params }: { params: Promise<Params> }) => {
  const { id } = React.use(params);
  const t = useTranslations("QueueLogsPage");
  useEffect(() => {
    document.title = `${t('title')} - ${id}`;
  }, [id]);

  const searchParams = useSearchParams()
  const logEndRef = useRef<HTMLDivElement>(null);
  const logType: QueueLogType = (searchParams.get('log') as QueueLogType) ?? 'video';

  const axiosPrivate = useAxiosPrivate()



  const { data, isPending, isError } = useGetQueueLogs(axiosPrivate, id, logType)

  useEffect(() => {
    const logScrollInterval = setInterval(() => {
      if (logEndRef.current) {
        logEndRef.current.scrollIntoView();
      }
    }, 1000);
    return () => clearInterval(logScrollInterval);
  });

  if (isPending) {
    return <GanymedeLoadingText message={t('loading')} />
  }

  if (isError) {
    return <div>{t('error')}</div>
  }

  return (
    <Box className={classes.logPage}>
      <div className={classes.logLine} dangerouslySetInnerHTML={{ __html: data }}></div>
      <div ref={logEndRef}></div>
    </Box>
  );
}

export default QueueLogsPage;