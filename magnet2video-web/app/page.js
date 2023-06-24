import Image from 'next/image'
import Navbar from './components/navbar/navbar'

export default function Home() {
  return (
    <section>
      <Navbar/>
      <div className='grid-container'>
        <div>
          <a>haojiahuo</a>
        </div>
        <a className='aside'/>
        <div>
          <a>haojiahuo</a>
        </div>
      </div>
    </section>
  )
}
