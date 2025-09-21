import { Channel } from "@/app/hooks/useChannels";
import { AspectRatio, Card, Center, Title, Image } from "@mantine/core";
import Link from "next/link";
import { env } from "next-runtime-env";
import classes from "./Card.module.css"

type Props = {
  channel: Channel
}

const ChannelCard = ({ channel }: Props) => {
  return (
    <div>
      <Link href={"/channels/" + channel.name} className={classes.link}>
        <Card key={channel.id} p="md" radius="md" >
          <AspectRatio ratio={300 / 300}>
            <Image src={`${(env('NEXT_PUBLIC_CDN_URL') ?? '')}${channel.image_path}`} alt={`${channel.name}`} fallbackSrc="/images/ganymede_default_channel_image.webp" />
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