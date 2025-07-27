import { Playlist, PlaylistGroupOperator, PlaylistRuleField, PlaylistRuleGroup, PlaylistRuleOperator, useGetPlaylistRules, useSavePlaylistRules, useTestPlaylistRules } from "@/app/hooks/usePlaylist";
import GanymedeLoadingText from "../utils/GanymedeLoadingText";
import { ActionIcon, Button, Divider, Flex, Text, Select, Group, Card, TextInput, NumberInput, Switch, Title, Box } from "@mantine/core";
import { useEffect, useState } from "react";
import { useAxiosPrivate } from "@/app/hooks/useAxios";
import { showNotification } from "@mantine/notifications";
import { IconTrash } from "@tabler/icons-react";
import { useTranslations } from "next-intl";
import { useForm } from "@mantine/form";

type Props = {
  playlist: Playlist
  handleClose: () => void;
}

interface SelectOption {
  label: string;
  value: string;
}

const PlaylistAutomationRuleModalContent = ({ playlist, handleClose }: Props) => {
  const t = useTranslations('PlaylistComponents')
  const axiosPrivate = useAxiosPrivate();
  const [saveButtonLoading, setSaveButtonLoading] = useState(false);

  const [testRuleVideoIdInput, setTestRuleVideoIdInput] = useState("");
  const [testRuleButtonLoading, setTestRuleButtonLoading] = useState(false);

  const { data: playlistRules, isPending: isPlaylistRulesPending, isError: isPlaylistRulesError } = useGetPlaylistRules(playlist.id);

  const savePlaylistRulesMutation = useSavePlaylistRules();
  const testPlaylistRulesMutation = useTestPlaylistRules();

  // Convert PlaylistRuleField, PlaylistRuleOperator, and PlaylistGroupOperator enums to SelectOptions
  // This allows us to use them in the Select components for field and operator selection
  const playlistRuleField: SelectOption[] = Object.entries(PlaylistRuleField).map(([key, value]) => ({
    label: key,
    value: value
  }));
  const playlistRuleOperator: SelectOption[] = Object.entries(PlaylistRuleOperator).map(([key, value]) => ({
    label: value === "contains" ? "Contains (case insensitive)" : key,
    value: value
  }));
  const operatorValues: SelectOption[] = Object.entries(PlaylistGroupOperator).map(([key, value]) => ({
    label: key,
    value: value
  }));

  const form = useForm<{ rule_groups: PlaylistRuleGroup[] }>({
    mode: "controlled",
    initialValues: {
      rule_groups: [],
    },
  });

  useEffect(() => {
    if (playlistRules) {
      form.setValues({
        rule_groups: playlistRules.map((group: any, groupIdx: number) => ({
          id: group.id,
          operator: group.operator,
          position: groupIdx,
          rules: (group.edges?.rules || []).map((rule: any, ruleIdx: number) => ({
            id: rule.id,
            name: rule.name,
            field: rule.field,
            operator: rule.operator,
            value: rule.value,
            position: rule.position ?? ruleIdx + 1,
            enabled: rule.enabled,
          })),
        })),
      });
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [playlistRules]);

  const handleTestRules = async () => {
    if (!testRuleVideoIdInput) {
      showNotification({
        message: t('automationRules.notifications.testEnterVideoId'),
        color: "red"
      });
      return;
    }
    setTestRuleButtonLoading(true);
    try {
      const response = await testPlaylistRulesMutation.mutateAsync({
        axiosPrivate,
        playlistId: playlist.id,
        videoId: testRuleVideoIdInput
      });

      if (response.data) {
        showNotification({
          title: t('automationRules.notifications.rulesMatchedTitle'),
          message: t('automationRules.notifications.rulesMatchedText'),
          color: "green"
        });
      } else {
        showNotification({
          title: t('automationRules.notifications.rulesNotMatchedTitle'),
          message: t('automationRules.notifications.rulesNotMatchedText'),
          color: "yellow"
        });
      }
    } catch (error) {
      console.error("Error testing rules:", error);
      showNotification({
        message: t('automationRules.notifications.errorText'),
        color: "red"
      });
    } finally {
      setTestRuleButtonLoading(false);
    }
  }

  const handleSubmitForm = async () => {
    const formValues = form.getValues()
    setSaveButtonLoading(true);
    try {
      await savePlaylistRulesMutation.mutateAsync({
        axiosPrivate,
        id: playlist.id,
        rules: formValues.rule_groups
      })

      showNotification({
        message: t('automationRules.notifications.saveText'),
        color: "green"
      });

      handleClose();
    } catch (error) {
      console.error(error);
      showNotification({
        message: t('automationRules.notifications.errorSaveText'),
        color: "red"
      });
    } finally {
      setSaveButtonLoading(false);
    }
  }

  if (isPlaylistRulesPending) {
    return <GanymedeLoadingText message={t('loading')} />;
  }

  if (isPlaylistRulesError) {
    return <div>{t('errorLoading')}</div>;
  }

  return (
    <Box py={"sm"}>
      <Title mb={"sm"} order={3}>
        {t.markup('automationRules.title', {
          playlistName: playlist.name,
        })}
      </Title>
      <Text>{t('automationRules.descriptionOne')}</Text>
      <Text>{t('automationRules.descriptionTwo')}</Text>
      <Text mb="xs">{t('automationRules.descriptionThree')}</Text>
      <form onSubmit={form.onSubmit(() => {
        handleSubmitForm()
      })}>
        <div>
          {form.values.rule_groups.map((group, groupIdx) => (
            <div key={group.id}>
              <Card shadow="sm" padding="md" radius="md" withBorder>
                <Group>
                  <Text>{t('automationRules.groupText')} {groupIdx + 1}</Text>
                  <ActionIcon
                    color="red"
                    onClick={() => {
                      form.setFieldValue(
                        "rule_groups",
                        form.values.rule_groups.filter((_, i) => i !== groupIdx)
                      );
                    }}
                  >
                    <IconTrash size={18} />
                  </ActionIcon>
                </Group>
                <Select
                  label={t('automationRules.groupOperatorLabel')}
                  data={operatorValues}
                  {...form.getInputProps(`rule_groups.${groupIdx}.operator`)}
                  style={{ width: 120 }}
                  withAsterisk
                />
                <Divider my="sm" />
                {group.rules.map((rule, ruleIdx) => (
                  <Card key={rule.id} shadow="xs" padding="sm" radius="sm" withBorder mb="sm">
                    <Group gap="xs">
                      <Text>{rule.name}</Text>
                      <ActionIcon
                        color="red"
                        onClick={() => {
                          const updatedRules = group.rules.filter((_, i) => i !== ruleIdx);
                          form.setFieldValue(
                            `rule_groups.${groupIdx}.rules`,
                            updatedRules
                          );
                        }}
                      >
                        <IconTrash size={16} />
                      </ActionIcon>
                    </Group>
                    <Group grow>
                      <TextInput
                        label={t('automationRules.nameLabel')}
                        {...form.getInputProps(`rule_groups.${groupIdx}.rules.${ruleIdx}.name`)}
                        withAsterisk
                      />
                      <NumberInput
                        label={t('automationRules.positionLabel')}
                        min={1}
                        {...form.getInputProps(`rule_groups.${groupIdx}.rules.${ruleIdx}.position`)}
                        withAsterisk
                        onBlur={() => {
                          const rules = [...form.values.rule_groups[groupIdx].rules];
                          const currentRule = rules.splice(ruleIdx, 1)[0];
                          let newPosition = Number(currentRule.position);

                          // Clamp position to valid range
                          if (newPosition < 1) newPosition = 1;
                          if (newPosition > rules.length + 1) newPosition = rules.length + 1;

                          // Insert current rule at new position
                          rules.splice(newPosition - 1, 0, currentRule);

                          // Reassign positions to be sequential
                          const updatedRules = rules.map((rule, idx) => ({
                            ...rule,
                            position: idx + 1,
                          }));

                          form.setFieldValue(`rule_groups.${groupIdx}.rules`, updatedRules);
                        }}
                      />
                      <Switch
                        mt={"lg"}
                        label={t('automationRules.enabledLabel')}
                        {...form.getInputProps(`rule_groups.${groupIdx}.rules.${ruleIdx}.enabled`, { type: 'checkbox' })}
                      />
                    </Group>
                    <Group grow>
                      <Select
                        label={t('automationRules.fieldLabel')}
                        data={playlistRuleField}
                        {...form.getInputProps(`rule_groups.${groupIdx}.rules.${ruleIdx}.field`)}
                        withAsterisk
                      />
                      <Select
                        label={t('automationRules.operatorLabel')}
                        data={playlistRuleOperator}
                        {...form.getInputProps(`rule_groups.${groupIdx}.rules.${ruleIdx}.operator`)}
                        withAsterisk
                      />
                      <TextInput
                        label={t('automationRules.valueLabel')}
                        {...form.getInputProps(`rule_groups.${groupIdx}.rules.${ruleIdx}.value`)}
                        withAsterisk
                      />
                    </Group>
                  </Card>
                ))}
                <Button
                  mt="sm"
                  variant="light"
                  onClick={() => {
                    const newRule = {
                      id: `rule-${Date.now()}`,
                      name: "",
                      field: "title",
                      operator: "contains",
                      value: "",
                      position: group.rules.length + 1,
                      enabled: true,
                    };
                    form.insertListItem(`rule_groups.${groupIdx}.rules`, newRule);
                  }}
                >
                  {t('automationRules.addRuleButton')}
                </Button>
              </Card>
              {(form.values.rule_groups.length > 1 && groupIdx < form.values.rule_groups.length - 1) && (
                <Divider my="lg" size="lg" label="OR" labelPosition="center" />
              )}
            </div>
          ))}
          <Group mt={"md"} gap="xs">
            <Button
              variant="outline"
              onClick={() => {
                const newGroup = {
                  id: `group-${Date.now()}`,
                  operator: "AND",
                  position: form.values.rule_groups.length,
                  rules: [],
                };
                form.insertListItem("rule_groups", newGroup);
              }}
            >
              {t('automationRules.addRuleGroupButton')}
            </Button>
            <Button type="submit" loading={saveButtonLoading} disabled={saveButtonLoading} color="green" aria-label="Save playlist rules">
              {t('automationRules.saveButton')}
            </Button>
          </Group>
          <Card mt={"md"}>
            <Title order={5}>{t('automationRules.testRulesTitle')}</Title>
            <Flex gap="xs"
              align="flex-end"
              direction="row"
              wrap="wrap">
              <TextInput
                label={t('automationRules.testRulesVideoIdInputLabel')}
                placeholder="1e5650a8-711e-4a8f-9266-21b8a55526d5"
                value={testRuleVideoIdInput}
                onChange={(e) => setTestRuleVideoIdInput(e.currentTarget.value)}
              />
              <Button loading={testRuleButtonLoading} onClick={handleTestRules}>
                {t('automationRules.testRulesButton')}
              </Button>
            </Flex>
          </Card>
        </div>
      </form>
    </Box>
  );
}

export default PlaylistAutomationRuleModalContent;