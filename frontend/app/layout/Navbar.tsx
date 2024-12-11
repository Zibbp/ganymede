"use client"
import { Menu, Group, Center, Burger, rem, Drawer, ScrollArea, Divider, Button, ActionIcon, TextInput, useMantineColorScheme, useComputedColorScheme } from '@mantine/core';
import { useDisclosure } from '@mantine/hooks';
import { IconChevronDown, IconMoon, IconSearch, IconSun, IconUserCircle } from '@tabler/icons-react';
import classes from './Navbar.module.css';
import useAuthStore from '../store/useAuthStore';
import Link from 'next/link';
import cx from "clsx"
import Image from 'next/image';
import { authLogout, UserRole } from '../hooks/useAuthentication';
import { useRouter } from 'next/navigation';

interface NavLink {
  link: string;
  label: string;
  auth?: boolean;
  role?: UserRole;
  links?: NavLink[];
}

const links: NavLink[] = [
  { link: '/', label: 'Home' },
  { link: '/channels', label: 'Channels' },
  {
    link: '/archive',
    label: 'Archive',
    auth: true,
    role: UserRole.Editor
  },
  { link: '/playlists', label: 'Playlists' },
  { link: '/queue', label: 'Queue', auth: true, role: UserRole.Editor },
  {
    link: '#',
    label: 'Admin',
    auth: true,
    role: UserRole.Admin,
    links: [
      { link: '/admin/channels', label: 'Channels' },
      { link: '/admin/watched', label: 'Watched Channels' },
      { link: '/admin/videos', label: 'Videos' },
      { link: '/admin/blocked-videos', label: 'Blocked Videos' },
      { link: '/admin/queue', label: 'Queue' },
      { link: '/admin/users', label: 'Users' },
      { link: '/admin/settings', label: 'Settings' },
      { link: '/admin/tasks', label: 'Tasks' },
      { link: '/admin/info', label: 'Information' },
    ],
  }
];

export function Navbar() {
  const { isLoggedIn, user, logout, hasPermission } = useAuthStore()

  const { setColorScheme } = useMantineColorScheme();
  const computedColorScheme = useComputedColorScheme('light', { getInitialValueInEffect: true });

  const router = useRouter()

  const [drawerOpened, { toggle: toggleDrawer, close: closeDrawer }] = useDisclosure(false);

  const handleLogout = async () => {
    try {
      await authLogout() // server side logout to clear session and cookies
      logout() // clear store
      router.push("/")

    } catch (error) {
      console.error("Error logging out", error)
    }
  }

  const items = links.filter(link => {
    // If link doesn't require auth, always show
    if (!link.auth) return true;

    // If link requires auth, check if logged in and has required role
    if (isLoggedIn && (!link.role || hasPermission(link.role))) {
      // If it's a dropdown menu, filter its items too
      if (link.links) {
        link.links = link.links.filter(subLink =>
          !subLink.auth || (isLoggedIn && (!subLink.role || hasPermission(subLink.role)))
        );

        // Only show the main menu if it has any visible sub-items
        return link.links.length > 0;
      }
      return true;
    }

    return false;
  }).map((link) => {
    const menuItems = link.links?.map((item) => (
      <Menu.Item key={item.label} component={Link} href={item.link}>
        {item.label}
      </Menu.Item>
    ));

    if (menuItems) {
      return (
        <Menu key={link.label} trigger="hover" transitionProps={{ exitDuration: 0 }} withinPortal>
          {/* @ts-expect-error fine */}
          <Menu.Target>
            <a
              href={link.link}
              className={classes.link}
              onClick={(event) => event.preventDefault()}
            >
              <Center>
                <span className={classes.linkLabel}>{link.label}</span>
                <IconChevronDown size="0.9rem" stroke={1.5} />
              </Center>
            </a>
          </Menu.Target>
          <Menu.Dropdown>{menuItems}</Menu.Dropdown>
        </Menu>
      );
    }

    return (
      <Link
        key={link.label}
        href={link.link}
        className={classes.link}
      >
        {link.label}
      </Link>
    );
  });

  return (
    <header className={classes.header}>
      <div className={classes.inner}>
        <Group gap={5} >
          <Image src="/images/ganymede_logo.png" height={32} width={32} alt="Ganymede logo" />
          <Group visibleFrom="sm">
            {items}
          </Group>
        </Group>
        <Group gap={5} visibleFrom="sm">

          <TextInput
            leftSectionPointerEvents="none"
            leftSection={<IconSearch style={{ width: rem(16), height: rem(16) }} stroke={1.5} />}
            placeholder="Search"
          />


          <ActionIcon
            onClick={() => setColorScheme(computedColorScheme === 'light' ? 'dark' : 'light')}
            variant="default"
            size="lg"
            aria-label="Toggle color scheme"
          >
            <IconSun className={cx(classes.icon, classes.light)} stroke={1.5} />
            <IconMoon className={cx(classes.icon, classes.dark)} stroke={1.5} />
          </ActionIcon>

          {isLoggedIn && (


            <Menu shadow="md" width={200}>
              {/* @ts-expect-error fine */}
              <Menu.Target>
                <ActionIcon variant="default" aria-label="Profile" size="lg">
                  <IconUserCircle style={{ width: '70%', height: '70%' }} stroke={1.5} />
                </ActionIcon>
              </Menu.Target>
              <Menu.Dropdown>
                <Menu.Label>{user?.username}</Menu.Label>
                <Menu.Item component={Link} href={`/profile`}>
                  Profile
                </Menu.Item>
                <Menu.Item onClick={handleLogout}>
                  Logout
                </Menu.Item>
              </Menu.Dropdown>
            </Menu>
          )}

          {!isLoggedIn && (
            <div>

              <Button component={Link} href="/login" variant="default" mr={5}>Log in</Button>
              <Button variant="default" >Sign up</Button>


            </div>
          )}

        </Group>
        <Burger opened={drawerOpened} onClick={toggleDrawer} size="sm" hiddenFrom="sm" />
      </div>

      <Drawer
        opened={drawerOpened}
        onClose={closeDrawer}
        size="100%"
        padding="md"
        title="Navigation"
        hiddenFrom="sm"
        zIndex={1000000}
      >
        <ScrollArea h={`calc(100vh - ${rem(80)})`} mx="-md">
          <Divider my="sm" />

          <a href="#" className={classes.link}>
            Home
          </a>



          <Divider my="sm" />

          <Group justify="center" grow pb="xl" px="md">
            <Button variant="default">Log in</Button>
            <Button>Sign up</Button>
          </Group>
        </ScrollArea>
      </Drawer>

    </header>
  );
}