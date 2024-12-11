import { Center, Stack } from "@mantine/core";
import GanymedeLoader from "./GanymedeLoader";

interface params {
  message: string;
}

const GanymedeLoadingText = ({ message }: params) => {
  return (
    <Center mt={10}>
      <Stack align="center">
        <GanymedeLoader />
        <div>{message}</div>
      </Stack>
    </Center>
  );
}

export default GanymedeLoadingText;