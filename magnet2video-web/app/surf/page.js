import styles from "./styles.module.css"
import FileList from "../components/fileitem"

async function getData() {
    const res = await fetch('http://101.35.200.143' + '/api/v2/list',{
        method: 'POST',
        // headers: {
        //     'X-RapidAPI-Key': 'your-rapidapi-key',
        //     'X-RapidAPI-Host': 'famous-quotes4.p.rapidapi.com',
        // },
    });
    // The return value is *not* serialized
    // You can return Date, Map, Set, etc.
   
    // Recommendation: handle errors
    if (!res.ok) {
      // This will activate the closest `error.js` Error Boundary
      throw new Error('Failed to fetch data');
    }
   
    return res.json();
}

async function Sufr(){
    const data = await getData();
    console.log(data);
    return (
        <>
        <div className={styles.header}>haojiahuo</div>
        <div className={styles.container}>
            <div className={styles.aside}>
                <FileList children={data}></FileList>
            </div>
            <div>
                <a>haojiahuo</a>
            </div>
        </div>
        </>
    )
}

export default Sufr