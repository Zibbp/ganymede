"use client";
import { usePathname, useRouter, useSearchParams } from "next/navigation";
import { FormEvent, useEffect, useRef, useState } from "react";
import { SearchField, useSearchVideos } from "../hooks/useVideos";
import { useVideoListParams } from "../hooks/useVideoListParams";
import useSettingsStore from "../store/useSettingsStore";
import GanymedeLoadingText from "../components/utils/GanymedeLoadingText";
import VideoGrid from "../components/videos/Grid";
import { Box, Button, Center, Collapse, Container, Group, rem, TextInput, Title, Text, Flex, Code } from "@mantine/core";
import { useTranslations } from "next-intl";
import { IconChevronDown, IconChevronUp, IconSearch, IconX } from "@tabler/icons-react";
import { useDisclosure } from "@mantine/hooks";


const SearchPage = () => {
  const router = useRouter();
  const pathname = usePathname();
  const searchParams = useSearchParams();
  const initialQ = searchParams.get("q") ?? "";

  const t = useTranslations("SearchPage");

  const [inputValue, setInputValue] = useState<string>(initialQ);
  const [searchTerm, setSearchTerm] = useState<string>(initialQ);

  const [advancedSearchOpened, { toggle: toggleAdvancedSearch }] = useDisclosure(false);

  useEffect(() => {
    document.title = `${t('title')} - ${searchTerm}`;
  }, [searchTerm, t]);

  const { page, videoTypes, sortBy, order, setPage, setVideoTypes, setSortBy, setOrder } = useVideoListParams();

  // Writes the search term to the URL while keeping the filter parameters, starting over at page 1
  const navigateToQuery = (q: string) => {
    const params = new URLSearchParams(searchParams.toString());
    params.set("q", q);
    params.delete("page");
    router.push(`${pathname}?${params.toString()}`);
  };

  const videoLimit = useSettingsStore((s) => s.videoLimit);
  const setVideoLimit = useSettingsStore((s) => s.setVideoLimit);

  const parseQuery = (q: SearchField) => {
    const m = q.match(/^(\w+):(.+)$/);
    console.log("parsedQuery", q, m);
    return m
      ? { field: m[1] as SearchField, query: m[2] }
      : { field: "title" as SearchField, query: q };
  };
  const { field, query } = parseQuery(searchTerm as SearchField);

  const { data: videos, isPending, isError } = useSearchVideos({
    limit: videoLimit,
    offset: (page - 1) * videoLimit,
    query,
    types: videoTypes,
    fields: [field],
    sort_by: sortBy,
    order: order,
  });

  const handleKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === "Enter") {
      navigateToQuery(inputValue);
      setSearchTerm(inputValue);
    }
  };
  const handleSubmit = (e: FormEvent<HTMLInputElement>) => {
    e.preventDefault();
    navigateToQuery(inputValue);
    setSearchTerm(inputValue);
  }

  if (isPending) return <GanymedeLoadingText message={t('loading')} />;
  if (isError) return <div>{t('error')}</div>;

  return (
    <div>
      <Center mt={10}><Title>{t('title')}</Title></Center>
      <Center>
        <TextInput
          style={{ width: '100%', maxWidth: 400 }}
          value={inputValue}
          onChange={(e) => setInputValue(e.currentTarget.value)}
          onKeyDown={handleKeyDown}
          onSubmit={handleSubmit}
          enterKeyHint="search"
          placeholder={t('searchInputPlaceholder')}
          leftSection={<IconSearch stroke={1.5} style={{ width: rem(16), height: rem(16) }} />}
          rightSection={
            inputValue && (
              <IconX
                stroke={1.5}
                style={{ width: rem(16), height: rem(16), cursor: 'pointer' }}
                onClick={() => { setInputValue(""); setSearchTerm(""); navigateToQuery(""); }}
              />
            )
          }
          rightSectionPointerEvents="auto"
        />
      </Center>
      <Center>
        <Box maw={700} mx="auto">
          <Group justify="center" mb={5} onClick={toggleAdvancedSearch}>
            <Flex
              justify="center"
              align="center"
              direction="row"
              wrap="wrap">
              <Button variant="transparent" size="compact-xs" radius="xs">
                <Text mr={5}>Advanced Search</Text>
                {advancedSearchOpened ? (
                  <IconChevronUp style={{ width: '15px', height: '15px' }} />
                ) : (
                  <IconChevronDown style={{ width: '15px', height: '15px' }} />)}

              </Button>

            </Flex>
          </Group>

          <Collapse expanded={advancedSearchOpened}>
            <Text>{t('advancedSearchText1')}</Text>
            <Text>{t.rich('advancedSearchText2', {
              code: (chunks) => <Code>{chunks}</Code>,
            })}</Text>
            <Text>{t('advancedSearchText3')}</Text>
            <ul>
              {Object.values(SearchField).map((field) => (
                <li key={field}>{field}</li>
              ))}
            </ul>
            <Text>{t('advancedSearchText4')}</Text>
            <ul>
              <li>
                <Text>{t.rich('advancedSearchTextExample1', {
                  code: (chunks) => <Code>{chunks}</Code>,
                })}</Text>
              </li>
              <li>
                <Text>{t.rich('advancedSearchTextExample2', {
                  code: (chunks) => <Code>{chunks}</Code>,
                })}</Text>
              </li>
            </ul>
            <Text>{t('advancedSearchText5')}</Text>
          </Collapse>
        </Box>
      </Center>


      <Container size="xl" px="xl" fluid={true}>
        <VideoGrid
          videos={videos.data}
          totalCount={videos.total_count}
          totalPages={videos.pages}
          currentPage={page}
          onPageChange={setPage}
          isPending={isPending}
          videoLimit={videoLimit}
          onVideoLimitChange={setVideoLimit}
          videoTypes={videoTypes}
          onVideoTypeChange={setVideoTypes}
          sortBy={sortBy}
          onSortByChange={setSortBy}
          order={order}
          onOrderChange={setOrder}
          showChannel={true}
        />
      </Container>
    </div>
  );
};

export default SearchPage;