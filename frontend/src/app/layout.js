import Navbar from './component/Navbar/Navbar'
import Footer from './component/Footer/Footer'
import AuthProvider from './component/AuthProvider/AuthProvider'
import './globals.css'
import { Inter } from 'next/font/google'
import { ThemeProvider } from './context/ThemeContext'


const inter = Inter({ subsets: ['latin'] })

export const metadata = {
  title: 'Create Next App',
  description: 'Generated by create next app',
}

export default function RootLayout({ children }) {
  return (
    <html lang="en">
      <body>
        <ThemeProvider>
            <AuthProvider>
              {/* <Script src="https://cdn.jsdelivr.net/npm/webtorrent/webtorrent.min.js" strategy="afterInteractive" onLoad={()=>{console.log("haojiahuo")}}> */}
                <div className="container">
                    <Navbar/>
                      {children}
                    <Footer/>
                  </div>
              {/* </Script> */}
            </AuthProvider>
        </ThemeProvider>      
      </body>
    </html>
  )
}