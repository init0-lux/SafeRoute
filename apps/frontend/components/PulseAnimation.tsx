import React, { useEffect, useRef } from 'react';
import { Animated, ViewStyle, StyleProp } from 'react-native';

interface PulseAnimationProps {
    children: React.ReactNode;
    duration?: number;
    minScale?: number;
    maxScale?: number;
    style?: StyleProp<ViewStyle>;
    active?: boolean;
}

export default function PulseAnimation({
    children,
    duration = 1000,
    minScale = 0.95,
    maxScale = 1.05,
    style,
    active = true,
}: PulseAnimationProps) {
    const scaleAnim = useRef(new Animated.Value(1)).current;

    useEffect(() => {
        if (!active) {
            scaleAnim.setValue(1);
            return;
        }

        const pulse = Animated.loop(
            Animated.sequence([
                Animated.timing(scaleAnim, {
                    toValue: maxScale,
                    duration: duration / 2,
                    useNativeDriver: true,
                }),
                Animated.timing(scaleAnim, {
                    toValue: minScale,
                    duration: duration / 2,
                    useNativeDriver: true,
                }),
            ]),
        );
        pulse.start();

        return () => pulse.stop();
    }, [active, duration, minScale, maxScale]);

    return (
        <Animated.View style={[style, { transform: [{ scale: scaleAnim }] }]}>
            {children}
        </Animated.View>
    );
}
