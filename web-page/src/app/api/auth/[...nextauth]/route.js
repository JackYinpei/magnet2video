import NextAuth from "next-auth"
import CreditentialsProvider from "next-auth/providers/credentials"

const handler = NextAuth({
  // Configure one or more authentication providers
  providers: [
    CreditentialsProvider({
      id: 'credentials',
      name: 'Credentials',
      async authorize(credentials) {
        const resp = await fetch("http://101.35.200.143/goapi/v1/user/login", {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
          },
          body: JSON.stringify({
            username: credentials.username,
            password: credentials.password,
          }),
        }).then((res) => {
          console.log("后端返回成功", res)
          return res.json()
        }).catch((err) => {
          console.log("登陆失败: ", err)
          return err
        })
        if (resp.status === 200000) {
          const token = resp.data.Authorization
          const name = resp.data.username
          console.log(token, "token", name, "username", resp, "resp")
          const user = {
            name: name,
            email: token,
            credentials: token,
          }
          return user
        }
        return null
      }
    })
  ],
  pages: {
    error: '/login',
  }

})

export {handler as GET, handler as POST}