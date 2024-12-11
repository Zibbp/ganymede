"use client"
import { useRouter } from 'next/navigation';
import useAuthStore from '@/app/store/useAuthStore';
import { useEffect } from 'react';

const OAuthPage = () => {
  const router = useRouter()
  const { fetchUser } = useAuthStore();

  useEffect(() => {

    const setUser = async () => {
      await fetchUser()
    }

    setUser().catch(console.error).then(() => router.push('/'))
  }, [fetchUser, router])

  return (
    <div></div>
  );
}

export default OAuthPage;