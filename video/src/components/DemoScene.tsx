import {
  AbsoluteFill,
  interpolate,
  spring,
  useCurrentFrame,
  useVideoConfig,
} from "remotion";
import { loadFont } from "@remotion/google-fonts/Inter";

const { fontFamily } = loadFont("normal", {
  weights: ["400", "500", "600", "700"],
  subsets: ["latin"],
});

type Props = {
  brandColor: string;
};

export const DemoScene: React.FC<Props> = ({ brandColor }) => {
  const frame = useCurrentFrame();
  const { fps, width, height } = useVideoConfig();

  const mockupScale = spring({
    frame,
    fps,
    config: { damping: 15, stiffness: 80 },
  });

  const mockupY = interpolate(mockupScale, [0, 1], [50, 0]);

  const scanlineY = interpolate(frame, [0, fps * 2], [0, 100], {
    extrapolateRight: "extend",
  }) % 100;

  const glowPulse = Math.sin(frame * 0.08) * 0.3 + 0.7;

  const isWide = width > height;
  const mockupWidth = isWide ? width * 0.7 : width * 0.9;

  return (
    <AbsoluteFill
      style={{
        fontFamily,
        justifyContent: "center",
        alignItems: "center",
      }}
    >
      {/* Floating label */}
      <div
        style={{
          position: "absolute",
          top: isWide ? 60 : 40,
          left: "50%",
          transform: "translateX(-50%)",
          opacity: interpolate(frame, [fps * 0.5, fps], [0, 1], {
            extrapolateLeft: "clamp",
            extrapolateRight: "clamp",
          }),
        }}
      >
        <span
          style={{
            fontSize: width * 0.018,
            color: brandColor,
            fontWeight: 600,
            textTransform: "uppercase",
            letterSpacing: "0.15em",
            background: `${brandColor}15`,
            padding: "12px 24px",
            borderRadius: 100,
            border: `1px solid ${brandColor}40`,
          }}
        >
          Beautiful Interface
        </span>
      </div>

      {/* Browser mockup */}
      <div
        style={{
          width: mockupWidth,
          transform: `scale(${mockupScale}) translateY(${mockupY}px)`,
          boxShadow: `0 40px 100px rgba(0,0,0,0.5), 0 0 ${60 * glowPulse}px ${brandColor}20`,
          borderRadius: 16,
          overflow: "hidden",
          background: "#1a1a2e",
        }}
      >
        {/* Browser chrome */}
        <div
          style={{
            height: 48,
            background: "#0f0f1a",
            display: "flex",
            alignItems: "center",
            padding: "0 16px",
            gap: 8,
          }}
        >
          <div style={{ width: 12, height: 12, borderRadius: "50%", background: "#ff5f57" }} />
          <div style={{ width: 12, height: 12, borderRadius: "50%", background: "#febc2e" }} />
          <div style={{ width: 12, height: 12, borderRadius: "50%", background: "#28c840" }} />
          <div
            style={{
              marginLeft: 16,
              flex: 1,
              height: 28,
              background: "#252538",
              borderRadius: 6,
              display: "flex",
              alignItems: "center",
              justifyContent: "center",
            }}
          >
            <span style={{ color: "#6b6b80", fontSize: 13 }}>
              cratedrop.local
            </span>
          </div>
        </div>

        {/* App UI mockup */}
        <div
          style={{
            height: isWide ? height * 0.5 : height * 0.4,
            background: "linear-gradient(180deg, #16162a 0%, #1a1a2e 100%)",
            position: "relative",
            overflow: "hidden",
          }}
        >
          {/* Scan line effect */}
          <div
            style={{
              position: "absolute",
              top: `${scanlineY}%`,
              left: 0,
              width: "100%",
              height: 2,
              background: `linear-gradient(90deg, transparent, ${brandColor}40, transparent)`,
              pointerEvents: "none",
            }}
          />

          {/* Sidebar */}
          <div
            style={{
              position: "absolute",
              left: 0,
              top: 0,
              bottom: 0,
              width: "18%",
              background: "#0f0f1a",
              borderRight: "1px solid #2a2a40",
              padding: 20,
            }}
          >
            <div
              style={{
                color: "white",
                fontSize: 16,
                fontWeight: 700,
                marginBottom: 24,
              }}
            >
              🎵 CrateDrop
            </div>
            {["Library", "Crates", "Upload", "Community"].map((item, i) => (
              <div
                key={item}
                style={{
                  padding: "10px 12px",
                  borderRadius: 8,
                  marginBottom: 4,
                  background: i === 0 ? `${brandColor}30` : "transparent",
                  color: i === 0 ? "white" : "#6b6b80",
                  fontSize: 14,
                  fontWeight: 500,
                }}
              >
                {item}
              </div>
            ))}
          </div>

          {/* Main content */}
          <div
            style={{
              marginLeft: "18%",
              padding: 24,
            }}
          >
            <h3
              style={{
                color: "white",
                fontSize: 24,
                fontWeight: 700,
                marginBottom: 20,
              }}
            >
              Your Library
            </h3>

            {/* Track list */}
            {[
              { title: "Deep House Mix Vol. 3", artist: "DJ Shadow", duration: "6:42" },
              { title: "Techno Warehouse", artist: "Producer X", duration: "8:15" },
              { title: "Summer Vibes 2024", artist: "Beach Collective", duration: "5:30" },
              { title: "Underground Bass", artist: "Bass Master", duration: "7:22" },
            ].map((track, i) => {
              const trackDelay = 0.5 + i * 0.15;
              const trackOpacity = interpolate(
                frame,
                [trackDelay * fps, (trackDelay + 0.3) * fps],
                [0, 1],
                { extrapolateLeft: "clamp", extrapolateRight: "clamp" }
              );
              const trackX = interpolate(
                frame,
                [trackDelay * fps, (trackDelay + 0.3) * fps],
                [20, 0],
                { extrapolateLeft: "clamp", extrapolateRight: "clamp" }
              );

              return (
                <div
                  key={i}
                  style={{
                    display: "flex",
                    alignItems: "center",
                    padding: "12px 16px",
                    background: i === 0 ? `${brandColor}15` : "transparent",
                    borderRadius: 8,
                    marginBottom: 8,
                    opacity: trackOpacity,
                    transform: `translateX(${trackX}px)`,
                    border: i === 0 ? `1px solid ${brandColor}40` : "1px solid transparent",
                  }}
                >
                  <div
                    style={{
                      width: 36,
                      height: 36,
                      background: `linear-gradient(135deg, ${brandColor}, #6366f1)`,
                      borderRadius: 6,
                      display: "flex",
                      justifyContent: "center",
                      alignItems: "center",
                      marginRight: 12,
                      fontSize: 14,
                    }}
                  >
                    {i === 0 ? "▶" : "♪"}
                  </div>
                  <div style={{ flex: 1 }}>
                    <div style={{ color: "white", fontSize: 14, fontWeight: 600 }}>
                      {track.title}
                    </div>
                    <div style={{ color: "#6b6b80", fontSize: 12 }}>{track.artist}</div>
                  </div>
                  <div style={{ color: "#6b6b80", fontSize: 13 }}>{track.duration}</div>
                </div>
              );
            })}
          </div>

          {/* Player bar */}
          <div
            style={{
              position: "absolute",
              bottom: 0,
              left: 0,
              right: 0,
              height: 64,
              background: "#0f0f1a",
              borderTop: "1px solid #2a2a40",
              display: "flex",
              alignItems: "center",
              padding: "0 20px",
            }}
          >
            <div
              style={{
                width: 44,
                height: 44,
                background: `linear-gradient(135deg, ${brandColor}, #6366f1)`,
                borderRadius: 8,
                marginRight: 12,
              }}
            />
            <div style={{ flex: 1 }}>
              <div style={{ color: "white", fontSize: 13, fontWeight: 600 }}>
                Deep House Mix Vol. 3
              </div>
              <div style={{ color: "#6b6b80", fontSize: 11 }}>DJ Shadow</div>
            </div>
            <div style={{ display: "flex", gap: 16, alignItems: "center" }}>
              <span style={{ color: "#6b6b80", fontSize: 18 }}>⏮</span>
              <div
                style={{
                  width: 40,
                  height: 40,
                  borderRadius: "50%",
                  background: brandColor,
                  display: "flex",
                  justifyContent: "center",
                  alignItems: "center",
                  color: "white",
                  fontSize: 16,
                }}
              >
                ▶
              </div>
              <span style={{ color: "#6b6b80", fontSize: 18 }}>⏭</span>
            </div>
          </div>
        </div>
      </div>
    </AbsoluteFill>
  );
};
