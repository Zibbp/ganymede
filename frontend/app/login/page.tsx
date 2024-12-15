'use client'
import { Container } from "@mantine/core";
import { AuthenticationForm, AuthFormType } from "../components/authentication/AuthenticationForm";
import { useEffect } from "react";

const Loginpage = () => {
  useEffect(() => {
    document.title = "Login";
  }, []);
  return (
    <div>
      <Container mt={25}>
        <AuthenticationForm type={AuthFormType.Login} />
      </Container>
    </div>
  );
}

export default Loginpage;