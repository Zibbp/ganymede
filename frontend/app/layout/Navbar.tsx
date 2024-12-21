"use client"
import { Menu, Group, Center, Burger, rem, Drawer, ScrollArea, Divider, Button, ActionIcon, TextInput, useMantineColorScheme, useComputedColorScheme, UnstyledButton, Collapse } from '@mantine/core';
import { useDisclosure } from '@mantine/hooks';
import { IconChevronDown, IconChevronUp, IconMoon, IconSearch, IconSun, IconUserCircle } from '@tabler/icons-react';
import classes from './Navbar.module.css';
import useAuthStore from '../store/useAuthStore';
import Link from 'next/link';
import cx from "clsx"
import Image from 'next/image';
import { authLogout, UserRole } from '../hooks/useAuthentication';
import { useRouter } from 'next/navigation';
import { useState } from 'react';

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
  const { isLoggedIn, user, logout, hasPermission } = useAuthStore();

  const { setColorScheme } = useMantineColorScheme();
  const computedColorScheme = useComputedColorScheme('light', { getInitialValueInEffect: true });

  const router = useRouter();

  const [drawerOpened, { toggle: toggleDrawer, close: closeDrawer }] = useDisclosure(false);
  const [adminLinksOpened, { toggle: toggleAdminLinks }] = useDisclosure(false);
  const [searchQuery, setSearchQuery] = useState("")

  const handleLogout = async () => {
    try {
      await authLogout(); // server-side logout to clear session and cookies
      logout(); // clear store
      router.push("/");
    } catch (error) {
      console.error("Error logging out", error);
    }
  };

  const filteredLinks = links.filter(link => {
    if (!link.auth) return true;
    if (isLoggedIn && (!link.role || hasPermission(link.role))) {
      if (link.links) {
        link.links = link.links.filter(subLink =>
          !subLink.auth || (isLoggedIn && (!subLink.role || hasPermission(subLink.role)))
        );
        return link.links.length > 0;
      }
      return true;
    }
    return false;
  });

  const renderLinks = (links: NavLink[], className: string) => {
    return links.map(link => {
      if (link.links) {
        return (
          <Menu key={link.label} trigger="hover" transitionProps={{ exitDuration: 0 }} withinPortal>
            {/* @ts-expect-error fine */}
            <Menu.Target>
              <a
                href={link.link}
                className={className}
                onClick={(event) => event.preventDefault()}
              >
                <Center>
                  <span className={classes.linkLabel}>{link.label}</span>
                  <IconChevronDown size="0.9rem" stroke={1.5} />
                </Center>
              </a>
            </Menu.Target>
            <Menu.Dropdown>
              {link.links.map(subLink => (
                <Menu.Item key={subLink.label} component={Link} href={subLink.link}>
                  {subLink.label}
                </Menu.Item>
              ))}
            </Menu.Dropdown>
          </Menu>
        );
      }

      return (
        <Link key={link.label} href={link.link} className={className}>
          {link.label}
        </Link>
      );
    });
  };

  const renderDrawerLinks = (links: NavLink[], className: string) => {
    return links.map(link => {
      if (link.links) {
        return (
          <div key={link.label}>
            <UnstyledButton
              onClick={toggleAdminLinks}
              className={cx(className, classes.collapseToggle)}
            >
              <Group>
                <span>{link.label}</span>
                {adminLinksOpened ? <IconChevronUp size="0.9rem" /> : <IconChevronDown size="0.9rem" />}
              </Group>
            </UnstyledButton>
            <Collapse in={adminLinksOpened}>
              <div className={classes.collapseContent}>
                {link.links.map(subLink => (
                  <Link key={subLink.label} href={subLink.link} className={classes.link}>
                    {subLink.label}
                  </Link>
                ))}
              </div>
            </Collapse>
          </div>
        );
      }

      return (
        <Link key={link.label} href={link.link} className={className}>
          {link.label}
        </Link>
      );
    });
  };

  const mainLinks = renderLinks(filteredLinks, classes.link);

  const drawerLinks = renderDrawerLinks(filteredLinks, classes.link);

  return (
    <header className={classes.header}>
      <div className={classes.inner}>
        <Group gap={5}>
          <Image src="/images/ganymede_logo.png" height={32} width={32} alt="Ganymede logo" />
          <Group visibleFrom="md">{mainLinks}</Group>
        </Group>
        <Group gap={5} visibleFrom="md">
          <TextInput
            value={searchQuery}
            onChange={(event) => setSearchQuery(event.currentTarget.value)}
            leftSectionPointerEvents="none"
            leftSection={<IconSearch style={{ width: rem(16), height: rem(16) }} stroke={1.5} />}
            placeholder="Search"
            onKeyUp={(e) => {
              if (e.key === "Enter") {
                router.push(`/search?q=${encodeURI(searchQuery)}`)
              }
            }}
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
          {isLoggedIn ? (
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
          ) : (
            <div>
              <Button component={Link} href="/login" variant="default" mr={5}>
                Log in
              </Button>
              <Button component={Link} href="/register" variant="default">Sign up</Button>
            </div>
          )}
        </Group>
        <Burger opened={drawerOpened} onClick={toggleDrawer} size="md" hiddenFrom="md" />
      </div>
      <Drawer
        opened={drawerOpened}
        onClose={closeDrawer}
        size="100%"
        padding="md"
        title="Navigation"
        hiddenFrom="md"
        zIndex={1000000}
      >
        <ScrollArea h={`calc(100vh - ${rem(80)})`} mx="-md">
          <Divider my="md" />
          {drawerLinks}
          <Divider my="md" />
          <Group justify="center" grow pb="xl" px="md">
            {isLoggedIn ? (
              <>
                <Button component={Link} href={`/profile`}>Profile</Button>
                <Button onClick={handleLogout}>Logout</Button>
              </>
            ) : (
              <>
                <Button component={Link} href="/login" variant="default">
                  Log in
                </Button>
                <Button component={Link} href="/register" variant="default">Sign up</Button>
              </>
            )}
          </Group>
        </ScrollArea>
      </Drawer>
    </header>
  );
}
