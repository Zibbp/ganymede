"use client"

import { Channel } from "@/app/hooks/useChannels";
import { AspectRatio, Card, Center, Title, Image, Skeleton } from "@mantine/core";
import Link from "next/link";
import { useState } from "react";
import { mediaSrc } from "@/app/util/util";
import classes from "./Card.module.css"

type Props = {
  channel: Channel
}

const ChannelCard = ({ channel }: Props) => {
  const [imageLoaded, setImageLoaded] = useState(false);
  const [prevImagePath, setPrevImagePath] = useState(channel.image_path);
  // Reset the loading skeleton when a new channel image arrives.
  if (channel.image_path !== prevImagePath) {
    setPrevImagePath(channel.image_path);
    setImageLoaded(false);
  }

  return (
    <div>
      <Link href={"/channels/" + channel.name} className={classes.link}>
        <Card key={channel.id} p="md" radius="md" >
          <AspectRatio ratio={300 / 300}>
            <Skeleton visible={!imageLoaded} animate radius="md">
              <Image
                src={mediaSrc(channel.image_path)}
                alt={`${channel.name}`}
                fallbackSrc="/images/ganymede_default_channel_image.webp"
                onLoad={() => setImageLoaded(true)}
                onError={() => setImageLoaded(true)}
              />
            </Skeleton>
          </AspectRatio>
          <Center mt={5}>
            <Title order={3} mt={5}>
              {channel.display_name}
            </Title>
          </Center>
        </Card>
      </Link>
    </div>
  );
}

export default ChannelCard;