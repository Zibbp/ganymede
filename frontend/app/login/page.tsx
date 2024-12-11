'use client'
import { Container } from "@mantine/core";
import { AuthenticationForm } from "../components/authentication/AuthenticationForm";

const Loginpage = () => {
  return (
    <div>
      <Container>
        <AuthenticationForm />
      </Container>
    </div>
  );
}

export default Loginpage;