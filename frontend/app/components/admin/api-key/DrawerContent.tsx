import {
  API_KEY_RESOURCE_META,
  API_KEY_SCOPES_CATALOG,
  ApiKeyResource,
  ApiKeyScope,
  ApiKeyTier,
  makeScope,
  useCreateApiKey,
} from "@/app/hooks/useApiKeys";
import { useAxiosPrivate } from "@/app/hooks/useAxios";
import {
  Button,
  Group,
  MultiSelect,
  Stack,
  Text,
  Textarea,
  TextInput,
} from "@mantine/core";
import { useForm, zodResolver } from "@mantine/form";
import { showNotification } from "@mantine/notifications";
import { useTranslations } from "next-intl";
import { useState } from "react";
import { z } from "zod";

type Props = {
  // onCreated is invoked with the freshly minted secret. The parent page
  // is responsible for showing it in the show-once modal — the secret
  // is intentionally not stored anywhere here.
  onCreated: (secret: string) => void;
};

// Build the MultiSelect data once. Mantine's grouped form is
// { group: "label", items: [{value, label}] }; resources whose routes
// are not yet enforcing scopes are tagged in the option label so the
// admin sees they're reserved for future migrations.
const buildScopeOptions = () => {
  const groups: { group: string; items: { value: string; label: string }[] }[] =
    [];
  for (const resource of Object.values(ApiKeyResource) as ApiKeyResource[]) {
    const meta = API_KEY_RESOURCE_META[resource];
    const items = (Object.values(ApiKeyTier) as ApiKeyTier[]).map((tier) => ({
      value: makeScope(resource, tier),
      label: meta.enforced
        ? makeScope(resource, tier)
        : `${makeScope(resource, tier)} (reserved)`,
    }));
    groups.push({
      group: meta.enforced ? meta.label : `${meta.label} (reserved)`,
      items,
    });
  }
  return groups;
};

const AdminApiKeyDrawerContent = ({ onCreated }: Props) => {
  const t = useTranslations("AdminApiKeyComponents");
  const axiosPrivate = useAxiosPrivate();
  const createApiKey = useCreateApiKey();
  const [loading, setLoading] = useState(false);

  // Bounds match the backend validator on CreateApiKeyRequest plus
  // strict membership in the catalog so the user can't paste a typo.
  const schema = z.object({
    name: z
      .string()
      .min(3, { message: t("validation.name") })
      .max(50, { message: t("validation.name") }),
    description: z.string().max(500, { message: t("validation.description") }),
    scopes: z
      .array(
        z
          .string()
          .refine(
            (s): s is ApiKeyScope =>
              (API_KEY_SCOPES_CATALOG as string[]).includes(s),
            { message: t("validation.scopes.unknown") }
          )
      )
      .min(1, { message: t("validation.scopes.required") }),
  });

  const form = useForm({
    mode: "controlled",
    initialValues: {
      name: "",
      description: "",
      scopes: [] as ApiKeyScope[],
    },
    validate: zodResolver(schema),
  });

  const scopeOptions = buildScopeOptions();

  // Quick-pick presets reset the field to a single canonical scope so
  // the admin doesn't have to find them in the grouped MultiSelect.
  const setPreset = (scopes: ApiKeyScope[]) => form.setFieldValue("scopes", scopes);

  const handleSubmit = async () => {
    try {
      setLoading(true);
      const result = await createApiKey.mutateAsync({
        axiosPrivate,
        input: form.getValues(),
      });
      showNotification({ message: t("createNotification") });
      // Hand the secret straight to the parent's show-once modal; never
      // persist it in component state, query cache, or local storage.
      onCreated(result.secret);
    } catch (error) {
      console.error(error);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div>
      <form onSubmit={form.onSubmit(handleSubmit)}>
        <Stack gap="sm">
          <TextInput
            withAsterisk
            label={t("nameLabel")}
            placeholder={t("namePlaceholder")}
            key={form.key("name")}
            {...form.getInputProps("name")}
          />

          <Textarea
            label={t("descriptionLabel")}
            placeholder={t("descriptionPlaceholder")}
            autosize
            minRows={2}
            key={form.key("description")}
            {...form.getInputProps("description")}
          />

          <div>
            <Text size="sm" fw={500} mb={4}>
              {t("scopeLabel")}{" "}
              <Text span c="red">
                *
              </Text>
            </Text>
            <Text size="xs" c="dimmed" mb="xs">
              {t("scopeDescription")}
            </Text>
            <Group gap="xs" mb="xs">
              <Button
                size="xs"
                variant="light"
                onClick={() => setPreset(["*:admin"])}
              >
                {t("presets.fullAdmin")}
              </Button>
              <Button
                size="xs"
                variant="light"
                onClick={() => setPreset(["*:read"])}
              >
                {t("presets.readAll")}
              </Button>
              <Button
                size="xs"
                variant="subtle"
                color="gray"
                onClick={() => setPreset([])}
              >
                {t("presets.clear")}
              </Button>
            </Group>
            <MultiSelect
              data={scopeOptions}
              searchable
              clearable
              hidePickedOptions
              placeholder={t("scopePlaceholder")}
              key={form.key("scopes")}
              {...form.getInputProps("scopes")}
            />
          </div>

          <Button mt={5} type="submit" loading={loading} fullWidth>
            {t("createButton")}
          </Button>
        </Stack>
      </form>
    </div>
  );
};

export default AdminApiKeyDrawerContent;
