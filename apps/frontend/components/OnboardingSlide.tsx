import React from 'react';
import { View, Text, StyleSheet, Dimensions } from 'react-native';
import { Colors, FontSizes, Spacing } from '@/constants/theme';

const { width, height } = Dimensions.get('window');

interface OnboardingSlideProps {
    icon: React.ReactNode;
    title: string;
    highlightedWords: string[];
    subtitle: string;
    accentColor: string;
}

export default function OnboardingSlide({
    icon,
    title,
    highlightedWords,
    subtitle,
    accentColor,
}: OnboardingSlideProps) {
    const renderTitle = () => {
        const words = title.split(' ');
        return words.map((word, index) => {
            const isHighlighted = highlightedWords.some(
                (hw) => hw.toLowerCase() === word.toLowerCase() || hw.toLowerCase().includes(word.toLowerCase())
            );
            return (
                <Text
                    key={index}
                    style={[
                        styles.titleWord,
                        isHighlighted && { color: accentColor },
                    ]}
                >
                    {word}{' '}
                </Text>
            );
        });
    };

    return (
        <View style={styles.container}>
            <View style={styles.iconContainer}>{icon}</View>
            <View style={styles.textContainer}>
                <Text style={styles.title}>{renderTitle()}</Text>
                <Text style={styles.subtitle}>{subtitle}</Text>
            </View>
        </View>
    );
}

const styles = StyleSheet.create({
    container: {
        width,
        height,
        backgroundColor: Colors.bgDark,
        justifyContent: 'flex-end',
        paddingBottom: 120,
    },
    iconContainer: {
        flex: 1,
        justifyContent: 'center',
        alignItems: 'flex-end',
        paddingRight: Spacing.xl,
        paddingTop: 80,
    },
    textContainer: {
        paddingHorizontal: Spacing.xl,
        paddingBottom: Spacing.xl,
    },
    title: {
        fontSize: FontSizes.hero,
        fontWeight: '800',
        color: Colors.white,
        lineHeight: FontSizes.hero * 1.15,
        marginBottom: Spacing.md,
        flexDirection: 'row',
        flexWrap: 'wrap',
    },
    titleWord: {
        fontSize: FontSizes.hero,
        fontWeight: '800',
        color: Colors.white,
        lineHeight: FontSizes.hero * 1.15,
    },
    subtitle: {
        fontSize: FontSizes.md,
        color: Colors.grayLight,
        lineHeight: FontSizes.md * 1.5,
        marginTop: Spacing.sm,
    },
});
