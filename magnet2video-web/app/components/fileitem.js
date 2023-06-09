function FileItem({children}){
    return (
        <li>{children}</li>
    )
}

function FileList({children}){
    return (
        <div className="file-list">
            <ol>
            {Object.keys(children).map((key) => (
                <FileItem children={children[key]}></FileItem>
            ))}
            </ol>
        </div>
    )
}

export default FileList