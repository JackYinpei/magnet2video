"use client"


import { useSession } from 'next-auth/react'
export default function Home() {

  const {data, status} = useSession()
  console.log(data, "session")
  
  return (
    <div>haojiahuo</div>
  )
}
