'use client'
import { Container } from "@mantine/core";
import { AuthenticationForm, AuthFormType } from "../components/authentication/AuthenticationForm";
import { useEffect } from "react";

const Loginpage = () => {
  useEffect(() => {
    document.title = "Register";
  }, []);
  return (
    <div>
      <Container mt={25}>
        <AuthenticationForm type={AuthFormType.Register} />
      </Container>
    </div>
  );
}

export default Loginpage;