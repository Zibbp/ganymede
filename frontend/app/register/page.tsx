'use client'
import { Container } from "@mantine/core";
import { AuthenticationForm, AuthFormType } from "../components/authentication/AuthenticationForm";

const Loginpage = () => {
  return (
    <div>
      <Container mt={25}>
        <AuthenticationForm type={AuthFormType.Register} />
      </Container>
    </div>
  );
}

export default Loginpage;