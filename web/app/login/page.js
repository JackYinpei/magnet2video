import useSWR from 'swr'

const fetcher = (url) => fetch(url).then((res) => res.json())


export default async function login() {
    const { data, error } = useSWR('/api/v1/login', fetcher)

    return (
        <div>
            <h1>haojiahuo</h1>
        </div>
    )
};