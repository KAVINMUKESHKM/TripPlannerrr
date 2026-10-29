import Image from "next/image";
import Link from "next/link";
import Button from "@mui/material/Button";
import Card from "@mui/material/Card";
import CardContent from "@mui/material/CardContent";
import { SiteHeader } from "@/components/site-header";
import { TripSearchForm } from "@/components/trip-search-form";

const ideas = [
  { title: "Coast", detail: "Small harbors and time by the water.", destination: "Goa" },
  { title: "Mountains", detail: "Walks and viewpoints worth the climb.", destination: "Manali" },
  { title: "Culture", detail: "Old streets, museums, and local markets.", destination: "Jaipur" },
];

export default function HomePage() {
  return (
    <main className="home-shell">
      <SiteHeader />
      <section className="hero">
        <div className="hero-copy">
          <p className="hero-kicker">A calmer way to plan</p>
          <h1>More journey.<br />Less juggling.</h1>
          <p className="hero-subtitle">Choose a place, gather a few good stops, and let the days take shape.</p>
          <Button href="/plan" variant="contained" size="large">Start planning</Button>
        </div>
        <div className="hero-visual"><Image className="hero-image" src="/assets/seaside.jpg" alt="Quiet coast with ocean and green cliffs" fill priority sizes="(max-width: 760px) 100vw, 42vw" /></div>
      </section>

      <section className="search-section" aria-labelledby="search-heading">
        <div className="section-intro">
          <div><span className="eyebrow">Start with a place</span><h2 id="search-heading">Where are you headed?</h2></div>
          <p>We’ll help you find good stops and make a day of them.</p>
        </div>
        <TripSearchForm />
      </section>

      <section className="ideas-section" aria-labelledby="ideas-heading">
        <div className="section-intro ideas-heading"><div><span className="eyebrow">A few ways in</span><h2 id="ideas-heading">Let curiosity set the route.</h2></div><Link className="quiet-link" href="/plan">Plan your own trip <span aria-hidden="true">↗</span></Link></div>
        <div className="idea-grid">{ideas.map((idea) => <Card component="a" className="idea-card" key={idea.title} href={`/plan?${new URLSearchParams({ destination: idea.destination, days: "3" })}`} variant="outlined">
          <CardContent><span className="idea-destination">{idea.destination}</span><h3>{idea.title}</h3><p>{idea.detail}</p><span className="idea-action">Explore destination <span aria-hidden="true">↗</span></span></CardContent>
        </Card>)}</div>
      </section>

      <footer className="site-footer"><Link className="brand" href="/"><span className="brand-mark">T</span><span>Trip Planner</span></Link><p>Leave a little space for the unexpected.</p><Link href="/trips">Your saved trips</Link></footer>
    </main>
  );
}
