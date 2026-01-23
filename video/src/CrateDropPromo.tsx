import { AbsoluteFill, Sequence, useVideoConfig } from "remotion";
import { TransitionSeries, linearTiming } from "@remotion/transitions";
import { fade } from "@remotion/transitions/fade";
import { slide } from "@remotion/transitions/slide";
import { IntroScene } from "./components/IntroScene";
import { ProblemScene } from "./components/ProblemScene";
import { FeatureScene } from "./components/FeatureScene";
import { DemoScene } from "./components/DemoScene";
import { OutroScene } from "./components/OutroScene";

export type PromoProps = {
  brandColor: string;
  accentColor: string;
};

export const CrateDropPromo: React.FC<PromoProps> = ({
  brandColor,
  accentColor,
}) => {
  const { fps } = useVideoConfig();

  return (
    <AbsoluteFill
      style={{
        background: "linear-gradient(135deg, #0f0f23 0%, #1a1a2e 50%, #16213e 100%)",
      }}
    >
      <TransitionSeries>
        {/* Intro - Logo reveal */}
        <TransitionSeries.Sequence durationInFrames={3 * fps}>
          <IntroScene brandColor={brandColor} />
        </TransitionSeries.Sequence>

        <TransitionSeries.Transition
          presentation={fade()}
          timing={linearTiming({ durationInFrames: Math.round(0.5 * fps) })}
        />

        {/* Problem - DJ frustrations */}
        <TransitionSeries.Sequence durationInFrames={3.5 * fps}>
          <ProblemScene />
        </TransitionSeries.Sequence>

        <TransitionSeries.Transition
          presentation={slide({ direction: "from-right" })}
          timing={linearTiming({ durationInFrames: Math.round(0.4 * fps) })}
        />

        {/* Features showcase */}
        <TransitionSeries.Sequence durationInFrames={4 * fps}>
          <FeatureScene brandColor={brandColor} accentColor={accentColor} />
        </TransitionSeries.Sequence>

        <TransitionSeries.Transition
          presentation={fade()}
          timing={linearTiming({ durationInFrames: Math.round(0.5 * fps) })}
        />

        {/* Demo/Screenshots */}
        <TransitionSeries.Sequence durationInFrames={3 * fps}>
          <DemoScene brandColor={brandColor} />
        </TransitionSeries.Sequence>

        <TransitionSeries.Transition
          presentation={fade()}
          timing={linearTiming({ durationInFrames: Math.round(0.5 * fps) })}
        />

        {/* Outro - CTA */}
        <TransitionSeries.Sequence durationInFrames={2.5 * fps}>
          <OutroScene brandColor={brandColor} accentColor={accentColor} />
        </TransitionSeries.Sequence>
      </TransitionSeries>
    </AbsoluteFill>
  );
};
