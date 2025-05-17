'use client'
import { Container } from "@mantine/core";
import { AuthenticationForm, AuthFormType } from "../components/authentication/AuthenticationForm";
import { useEffect } from "react";
import { useTranslations } from "next-intl";
import { usePageTitle } from "../util/util";

const RegisterPage = () => {
  const t = useTranslations("AuthenticationPages")
  usePageTitle(t('registerPageTitle'))
  return (
    <div>
      <Container mt={25}>
        <AuthenticationForm type={AuthFormType.Register} />
      </Container>
    </div>
  );
}

export default RegisterPage;