"use client"
import { Container, Group, } from '@mantine/core';
import classes from './Hero.module.css';

export function LandingHero() {
  return (
    <div className={classes.root}>
      <Container size="lg">
        <Group justify="space-between">

          <div>
            foo
          </div>

          <div>
            bar
          </div>
        </Group>
        {/* <div className={classes.inner}>
          <div className={classes.content}>
            <Title className={classes.title}>
              A{' '}
              <Text
                component="span"
                inherit
                variant="gradient"
                gradient={{ from: 'pink', to: 'yellow' }}
              >
                fully featured
              </Text>{' '}
              React components library
            </Title>

            <Text className={classes.description} mt={30}>
              Build fully functional accessible web applications with ease – Mantine includes more
              than 100 customizable components and hooks to cover you in any situation
            </Text>

            <Button
              variant="gradient"
              gradient={{ from: 'pink', to: 'yellow' }}
              size="xl"
              className={classes.control}
              mt={40}
            >
              Get started
            </Button>
          </div>
        </div> */}
      </Container>
    </div>
  );
}