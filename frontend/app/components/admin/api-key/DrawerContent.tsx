import { ApiKeyScope, useCreateApiKey } from "@/app/hooks/useApiKeys";
import { useAxiosPrivate } from "@/app/hooks/useAxios";
import { Button, Select, Textarea, TextInput } from "@mantine/core";
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

const AdminApiKeyDrawerContent = ({ onCreated }: Props) => {
  const t = useTranslations("AdminApiKeyComponents");
  const axiosPrivate = useAxiosPrivate();
  const createApiKey = useCreateApiKey();
  const [loading, setLoading] = useState(false);

  // Bounds match the backend validator on CreateApiKeyRequest.
  const schema = z.object({
    name: z
      .string()
      .min(3, { message: t("validation.name") })
      .max(50, { message: t("validation.name") }),
    description: z.string().max(500, { message: t("validation.description") }),
    scope: z.nativeEnum(ApiKeyScope),
  });

  const form = useForm({
    mode: "controlled",
    initialValues: {
      name: "",
      description: "",
      scope: ApiKeyScope.Read,
    },
    validate: zodResolver(schema),
  });

  const scopeOptions = Object.values(ApiKeyScope).map((s) => ({
    value: s,
    label: s.charAt(0).toUpperCase() + s.slice(1),
  }));

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
        <TextInput
          withAsterisk
          label={t("nameLabel")}
          placeholder={t("namePlaceholder")}
          key={form.key("name")}
          {...form.getInputProps("name")}
        />

        <Textarea
          mt="sm"
          label={t("descriptionLabel")}
          placeholder={t("descriptionPlaceholder")}
          autosize
          minRows={2}
          key={form.key("description")}
          {...form.getInputProps("description")}
        />

        <Select
          mt="sm"
          withAsterisk
          label={t("scopeLabel")}
          description={t("scopeDescription")}
          data={scopeOptions}
          allowDeselect={false}
          key={form.key("scope")}
          {...form.getInputProps("scope")}
        />

        <Button mt={15} type="submit" loading={loading} fullWidth>
          {t("createButton")}
        </Button>
      </form>
    </div>
  );
};

export default AdminApiKeyDrawerContent;
