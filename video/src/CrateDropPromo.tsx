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
        {/* Intro - Logo reveal */}
        <TransitionSeries.Sequence durationInFrames={Math.round(3.5 * fps)}>
          <IntroScene colors={colors} />
        </TransitionSeries.Sequence>

        <TransitionSeries.Transition
          presentation={fade()}
          timing={springTiming({ config: { damping: 200 }, durationInFrames: Math.round(0.6 * fps) })}
        />

        {/* Problem - DJ frustrations */}
        <TransitionSeries.Sequence durationInFrames={Math.round(3.5 * fps)}>
          <ProblemScene colors={colors} />
        </TransitionSeries.Sequence>

        <TransitionSeries.Transition
          presentation={slide({ direction: "from-right" })}
          timing={springTiming({ config: { damping: 200 }, durationInFrames: Math.round(0.5 * fps) })}
        />

        {/* Features showcase */}
        <TransitionSeries.Sequence durationInFrames={Math.round(3.5 * fps)}>
          <FeatureScene colors={colors} />
        </TransitionSeries.Sequence>

        <TransitionSeries.Transition
          presentation={fade()}
          timing={linearTiming({ durationInFrames: Math.round(0.4 * fps) })}
        />

        {/* SoundCloud Auto-Sync Feature */}
        <TransitionSeries.Sequence durationInFrames={Math.round(4 * fps)}>
          <SoundCloudScene colors={colors} />
        </TransitionSeries.Sequence>

        <TransitionSeries.Transition
          presentation={slide({ direction: "from-bottom" })}
          timing={springTiming({ config: { damping: 200 }, durationInFrames: Math.round(0.5 * fps) })}
        />

        {/* Demo/Screenshots */}
        <TransitionSeries.Sequence durationInFrames={Math.round(3 * fps)}>
          <DemoScene colors={colors} />
        </TransitionSeries.Sequence>

        <TransitionSeries.Transition
          presentation={fade()}
          timing={linearTiming({ durationInFrames: Math.round(0.5 * fps) })}
        />

        {/* Outro - CTA */}
        <TransitionSeries.Sequence durationInFrames={Math.round(2.5 * fps)}>
          <OutroScene colors={colors} />
        </TransitionSeries.Sequence>
      </TransitionSeries>
    </AbsoluteFill>
  );
};
