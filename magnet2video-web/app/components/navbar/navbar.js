import Link from "next/link";

const Navbar = () => {
  return (
    <nav>
      <ul
        style={{
          display: "grid",
          gridTemplateColumns: "repeat(3, 1fr)",
          gap: "1rem",
          listStyle: "none",
          padding: 0,
        //   center the nav items
            alignItems: 'center',
        //   justifyContent: 'center',
        }}
      >
        <li>
          <Link href="/">Home</Link>
        </li>
        <li>
          <Link href="/about">About</Link>
        </li>
        <li>
          <Link href="/contact">Contact</Link>
        </li>
      </ul>
    </nav>
  );
};

export default Navbar;
