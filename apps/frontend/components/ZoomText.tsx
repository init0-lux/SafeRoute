import React, { useEffect, useRef } from 'react';
import { Animated, TextStyle, StyleProp } from 'react-native';

interface ZoomTextProps {
    text: string;
    style?: StyleProp<TextStyle>;
    duration?: number;
    onComplete?: () => void;
}

export default function ZoomText({
    text,
    style,
    duration = 800,
    onComplete,
}: ZoomTextProps) {
    const scaleAnim = useRef(new Animated.Value(0.3)).current;
    const opacityAnim = useRef(new Animated.Value(0)).current;

    useEffect(() => {
        scaleAnim.setValue(0.3);
        opacityAnim.setValue(0);

        Animated.parallel([
            Animated.sequence([
                Animated.timing(scaleAnim, {
                    toValue: 1.3,
                    duration: duration * 0.5,
                    useNativeDriver: true,
                }),
                Animated.timing(scaleAnim, {
                    toValue: 1,
                    duration: duration * 0.3,
                    useNativeDriver: true,
                }),
                Animated.timing(scaleAnim, {
                    toValue: 0.8,
                    duration: duration * 0.2,
                    useNativeDriver: true,
                }),
            ]),
            Animated.sequence([
                Animated.timing(opacityAnim, {
                    toValue: 1,
                    duration: duration * 0.3,
                    useNativeDriver: true,
                }),
                Animated.delay(duration * 0.4),
                Animated.timing(opacityAnim, {
                    toValue: 0,
                    duration: duration * 0.3,
                    useNativeDriver: true,
                }),
            ]),
        ]).start(() => onComplete?.());
    }, [text]);

    return (
        <Animated.Text
            style={[
                style,
                {
                    transform: [{ scale: scaleAnim }],
                    opacity: opacityAnim,
                },
            ]}
        >
            {text}
        </Animated.Text>
    );
}
