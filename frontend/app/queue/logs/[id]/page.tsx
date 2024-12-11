"use client"
import GanymedeLoadingText from "@/app/components/utils/GanymedeLoadingText";
import { useAxiosPrivate } from "@/app/hooks/useAxios";
import { QueueLogType, useGetQueueLogs } from "@/app/hooks/useQueue";
import { Box } from "@mantine/core";
import { useSearchParams } from "next/navigation";
import React, { useEffect, useRef } from "react";
import classes from "./QueueLogsPage.module.css"

interface Params {
  id: string;
}

const QueueLogsPage = ({ params }: { params: Promise<Params> }) => {
  const { id } = React.use(params);
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
    return <GanymedeLoadingText message="Loading logs" />
  }

  if (isError) {
    return <div>Error loading logs</div>
  }

  return (
    <Box className={classes.logPage}>
      <div className={classes.logLine} dangerouslySetInnerHTML={{ __html: data }}></div>
      <div ref={logEndRef}></div>
    </Box>
  );
}

export default QueueLogsPage;