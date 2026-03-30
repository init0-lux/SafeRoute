import { View, Text, StyleSheet } from "react-native";

type Props = {
    title: string;
    description: string;
};

export default function InfoCard({ title, description }: Props) {
    return (
        <View style={styles.container}>
            <Text style={styles.title}>{title}</Text>
            <Text style={styles.description}>{description}</Text>
        </View>
    );
}

const styles = StyleSheet.create({
    container: {
        width: "100%",
        backgroundColor: "#22201F",

        borderTopLeftRadius: 40,
        borderTopRightRadius: 40,

        paddingHorizontal: 24,
        paddingTop: 20,
        paddingBottom: 40,

        marginTop: 20,

        alignItems: "center",

        // shadow (iOS)
        shadowColor: "#000",
        shadowOpacity: 0.1,
        shadowRadius: 20,
        shadowOffset: { width: 0, height: -10 },

        // shadow (Android)
        elevation: 10,
    },

    title: {
        color: "#ABC339",
        textAlign: "center",
        fontSize: 18,
        fontStyle: "italic",
        marginVertical: 12,
    },

    description: {
        color: "white",
        fontSize: 14,
        textAlign: "center",
        lineHeight: 20,
    },
});