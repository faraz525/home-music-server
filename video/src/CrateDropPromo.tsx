import { AbsoluteFill, useVideoConfig } from "remotion";
import { TransitionSeries, linearTiming, springTiming } from "@remotion/transitions";
import { fade } from "@remotion/transitions/fade";
import { slide } from "@remotion/transitions/slide";
import { IntroScene } from "./components/IntroScene";
import { ProblemScene } from "./components/ProblemScene";
import { FeatureScene } from "./components/FeatureScene";
import { SoundCloudScene } from "./components/SoundCloudScene";
import { DemoScene } from "./components/DemoScene";
import { OutroScene } from "./components/OutroScene";

export type CrateColors = {
  black: string;
  surface: string;
  elevated: string;
  border: string;
  amber: string;
  amberLight: string;
  amberDark: string;
  cyan: string;
  cyanDark: string;
  cream: string;
  muted: string;
  subtle: string;
};

export type PromoProps = {
  colors: CrateColors;
};

export const CrateDropPromo: React.FC<PromoProps> = ({ colors }) => {
  const { fps } = useVideoConfig();

  return (
    <AbsoluteFill style={{ backgroundColor: colors.black }}>
      <TransitionSeries>
        {/* Intro - Logo reveal (simple, 3.5s) */}
        <TransitionSeries.Sequence durationInFrames={Math.round(3.5 * fps)}>
          <IntroScene colors={colors} />
        </TransitionSeries.Sequence>

        <TransitionSeries.Transition
          presentation={fade()}
          timing={springTiming({ config: { damping: 200 }, durationInFrames: Math.round(0.6 * fps) })}
        />

        {/* Problem - 4 items with strike animation (6s) */}
        <TransitionSeries.Sequence durationInFrames={Math.round(6 * fps)}>
          <ProblemScene colors={colors} />
        </TransitionSeries.Sequence>

        <TransitionSeries.Transition
          presentation={slide({ direction: "from-right" })}
          timing={springTiming({ config: { damping: 200 }, durationInFrames: Math.round(0.5 * fps) })}
        />

        {/* Features - 4 cards with descriptions (5s) */}
        <TransitionSeries.Sequence durationInFrames={Math.round(5 * fps)}>
          <FeatureScene colors={colors} />
        </TransitionSeries.Sequence>

        <TransitionSeries.Transition
          presentation={fade()}
          timing={linearTiming({ durationInFrames: Math.round(0.4 * fps) })}
        />

        {/* SoundCloud Auto-Sync - sync visual + 3 tracks (6s) */}
        <TransitionSeries.Sequence durationInFrames={Math.round(6 * fps)}>
          <SoundCloudScene colors={colors} />
        </TransitionSeries.Sequence>

        <TransitionSeries.Transition
          presentation={slide({ direction: "from-bottom" })}
          timing={springTiming({ config: { damping: 200 }, durationInFrames: Math.round(0.5 * fps) })}
        />

        {/* Demo - browser mockup with tracks (4.5s) */}
        <TransitionSeries.Sequence durationInFrames={Math.round(4.5 * fps)}>
          <DemoScene colors={colors} />
        </TransitionSeries.Sequence>

        <TransitionSeries.Transition
          presentation={fade()}
          timing={linearTiming({ durationInFrames: Math.round(0.5 * fps) })}
        />

        {/* Outro - CTA (3.5s) */}
        <TransitionSeries.Sequence durationInFrames={Math.round(3.5 * fps)}>
          <OutroScene colors={colors} />
        </TransitionSeries.Sequence>
      </TransitionSeries>
    </AbsoluteFill>
  );
};
