import NextAuth from "next-auth"
import GoogleProvider from "next-auth/providers/google"
import CreditentialsProvider from "next-auth/providers/credentials"

const handler = NextAuth({
  // Configure one or more authentication providers
  providers: [
    CreditentialsProvider({
      id: 'credentials',
      name: 'Credentials',
      async authorize(credentials) {
        const uname = credentials.username
        const pass = credentials.password
        const email = credentials.email
        if (uname === "haojiahuo" && pass === "haojiahuo") {
          return {
            name: uname,
            email: email,
          }
        } else {
          throw new Error("Login failed")
        }
      }
    })
  ],

})

export {handler as GET, handler as POST}