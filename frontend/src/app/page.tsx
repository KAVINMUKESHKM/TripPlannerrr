import Image from "next/image";
import Link from "next/link";
import { SiteHeader } from "@/components/site-header";
import { TripSearchForm } from "@/components/trip-search-form";

const ideas = [
  { title: "Find the unhurried coast", detail: "Small harbors, open water, and long lunches.", image: "/assets/seaside.jpg", destination: "Goa" },
  { title: "Make room for the view", detail: "Walks and overlooks worth building a day around.", image: "/assets/mountains.jpg", destination: "Manali" },
  { title: "Follow the old streets", detail: "Museums, markets, and places with a story.", image: "/assets/cultural.jpg", destination: "Jaipur" },
];

export default function HomePage() {
  return (
    <main className="home-shell">
      <SiteHeader />
      <section className="hero">
        <Image className="hero-image" src="/assets/seaside.jpg" alt="Ocean waves meeting a secluded beach below green cliffs" fill priority sizes="100vw" />
        <div className="hero-shade" />
        <div className="hero-copy">
          <p className="hero-kicker">A little more room to wander</p>
          <h1>Make a trip<br />of your own.</h1>
          <p className="hero-subtitle">Find places worth stopping for. Put the days together in a way that feels right.</p>
        </div>
        <div className="hero-caption"><span className="caption-rule" />Somewhere between the plan and the place</div>
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
        <div className="idea-grid">{ideas.map((idea) => <Link className="idea-card" key={idea.title} href={`/plan?${new URLSearchParams({ destination: idea.destination, days: "3", date: "" })}`}>
          <div className="idea-photo"><Image src={idea.image} alt="" fill sizes="(max-width: 700px) 90vw, 33vw" /></div>
          <div className="idea-copy"><h3>{idea.title}</h3><p>{idea.detail}</p><span className="idea-destination">Try {idea.destination}</span></div>
        </Link>)}</div>
      </section>

      <footer className="site-footer"><Link className="brand" href="/"><span className="brand-mark">T</span><span>Trip Planner</span></Link><p>Leave a little space for the unexpected.</p><Link href="/trips">Your saved trips</Link></footer>
    </main>
  );
}
